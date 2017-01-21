package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gbl08ma/cachet"
	urapi "github.com/gbl08ma/uptimerobot-api"
	"github.com/gorilla/mux"
)

var configFilename string

var client *cachet.Client
var ur *urapi.UptimeRobot
var config *Config

var monitorStatus map[int]urapi.MonitorStatus
var metricLastSent map[int]time.Time

var pauseMonitoringChan chan bool
var resumeMonitoringChan chan struct{}

// UpdateComponentStatus updates the specified Cachet components to the
// specified status
func UpdateComponentStatus(componentStatus int, componentIDs ...int) {
	for _, id := range componentIDs {
		log.Println("Updating Cachet component", id,
			"to Cachet status", componentStatus)
		c := cachet.Component{
			ID:      id,
			Status:  componentStatus,
			Enabled: true,
		}
		_, _, err := client.Components.Update(id, &c)
		if err != nil {
			log.Println("UpdateComponentStatus:", err)
		}
	}
}

// UpdateMetric updates the specified Cachet metrics, setting their value for
// the specified time
func UpdateMetric(value int, timestamp time.Time, metricIDs ...int) {
	for _, id := range metricIDs {
		log.Println("Updating Cachet metric", id, "to value", value)

		_, _, err := client.Metrics.AddPoint(
			id, value, strconv.FormatInt(timestamp.Unix(), 10))

		if err != nil {
			log.Println("UpdateMetric:", err)
		}
	}
}

// UpdateComponents checks if the status of a UR monitor changed since the last
// check, and if yes, updates all Cachet components linked to that monitor.
func UpdateComponents(monitor urapi.Monitor) {
	// check if status changed
	if status, ok := monitorStatus[monitor.ID]; !ok ||
		monitor.Status != status {
		// status changed
		if components, cok := config.MonitorComponentMap[monitor.ID]; cok {
			status, statusok := config.MonitorComponentStatusMap[monitor.Status]
			if statusok {
				UpdateComponentStatus(status, components...)
			}
		}
		// update status
		monitorStatus[monitor.ID] = monitor.Status
	}
}

// UpdateMetrics updates all Cachet metrics associated with a UR monitor
func UpdateMetrics(monitor urapi.Monitor) {
	// update metric, if one is defined for this monitor
	if metrics, ok := config.MonitorMetricMap[monitor.ID]; ok {
		if len(monitor.ResponseTimes) > 0 {
			// first array entry is always (as far as we could see) the most
			// recent entry
			if time.Time(monitor.ResponseTimes[0].DateTime).After(metricLastSent[monitor.ID]) {
				UpdateMetric(monitor.ResponseTimes[0].Value,
					time.Time(monitor.ResponseTimes[0].DateTime), metrics...)
				metricLastSent[monitor.ID] =
					time.Time(monitor.ResponseTimes[0].DateTime)
			}
		}
	}
}

// Refresh retrieves monitor status from Uptime Robot and updates Cachet metrics
// and components, with the associations defined in the config.
func Refresh() error {
	monitors, err := ur.GetMonitors(&urapi.GetMonitorsInput{
		ResponseTimes: true,
	})
	if err != nil {
		log.Println("Refresh:", err)
		return err
	}
	for _, monitor := range monitors {
		UpdateComponents(monitor)
		UpdateMetrics(monitor)
	}
	return nil
}

// InitialSetup dumps to the console a help message about setting up Upcachet,
// and saves an example config file.
func InitialSetup() error {
	log.Println("No monitor-component map configured. To aid with " +
		"configuration, here is a list of monitors:")
	monitors, err := ur.GetMonitors(&urapi.GetMonitorsInput{
		ResponseTimes: true,
	})
	if err != nil {
		return err
	}
	for _, monitor := range monitors {
		log.Println(monitor.ID, "-", monitor.FriendlyName)
	}
	log.Println("and here is the list of Cachet components:")

	componentResponse, _, err := client.Components.GetAll()
	if err != nil {
		return err
	}
	for _, component := range componentResponse.Components {
		log.Println(component.ID, "-", component.Name)
	}

	config.MonitorComponentMap[123] = []int{456, 789}
	log.Println("To aid with configuration, an example mapping between "+
		"fictious Uptime Robot monitor 123 and fictious Cachet components "+
		"456 and 789 has been added to", configFilename)
	return config.Save(configFilename)
}

// MonitorUptimeRobot periodically updates UR monitor status and changes Cachet
// components and metrics accordingly
func MonitorUptimeRobot() error {
	waitInterval := 1 * time.Minute

	account, err := ur.GetAccountDetails()
	if err != nil {
		return fmt.Errorf("MonitorUptimeRobot: "+
			"error getting account details: %s", err)
	}
	log.Println("Uptime Robot account using",
		account.UpMonitors+account.DownMonitors+account.PausedMonitors,
		"monitors out of", account.MonitorLimit)

	if len(config.MonitorComponentMap) == 0 {
		return InitialSetup()
	}

	log.Println("UptimeRobot monitor interval is",
		account.MonitorInterval, "minutes.")
	if config.CheckInterval.Nanoseconds() == 0 {
		// make waitInterval be half the monitor interval
		waitInterval = time.Duration(account.MonitorInterval) * 30 * time.Second
		log.Println("Check interval is", waitInterval.Seconds(), "seconds.")
	} else {
		waitInterval = config.CheckInterval
	}

	ticker := time.NewTicker(waitInterval)

	Refresh()

	paused := false
	for {
		select {
		case <-ticker.C:
			if !paused {
				Refresh()
			}
		case exit := <-pauseMonitoringChan:
			if exit {
				return nil
			}
			paused = true
		case <-resumeMonitoringChan:
			paused = false
			Refresh()
		}
	}
}

// Index handles a request for the index
func Index(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Upcachet Uptime Robot endpoint")
}

func main() {
	flag.StringVar(&configFilename, "c", defaultConfigFilename,
		"config filename")
	flag.Parse()

	monitorStatus = make(map[int]urapi.MonitorStatus)
	metricLastSent = make(map[int]time.Time)
	pauseMonitoringChan = make(chan bool, 10)
	resumeMonitoringChan = make(chan struct{}, 10)
	log.Println("Upcachet - Uptime Robot -> Cachet bridge")
	log.Println("Version", version)
	log.Println("Copyright Segvault (http://segvault.tny.im) 2015-2017")
	log.Println("---")

	if f := os.Getenv("UPCACHET_CONFIG_FILE"); f != "" &&
		configFilename == defaultConfigFilename {
		configFilename = f
	}

	log.Print("Using config file: ", configFilename)
	config = NewConfig()
	// if this errors, there's no problem: we'll just use the default config
	err := config.Load(configFilename)
	if err != nil {
		log.Print("Config defaults loaded")
		config.Save(configFilename)
	}
	log.Print("Config loaded")

	if config.CachetEndpoint == "" {
		log.Println("Please specify a Cachet endpoint either through the " +
			"config file, or through environment variable " +
			"UPCACHET_CACHET_ENDPOINT")
		return
	}

	log.Print("Initializing Cachet client...")
	client, _ = cachet.NewClient(config.CachetEndpoint, nil)
	log.Print("Pinging Cachet API endpoint...")
	pong, resp, err := client.General.Ping()

	if err != nil {
		log.Println("Ping fail:", err)
		return
	}
	if resp.StatusCode != 200 {
		log.Println("Ping fail")
		log.Printf("Result: %s\n", pong)
		log.Printf("Status: %s\n", resp.Status)
		if resp.StatusCode != 200 {
			return
		}
	} else {
		log.Print("Ping success")
	}

	if config.CachetAPIkey == "" {
		log.Println("Please specify a Cachet API key either through the " +
			"config file, or through environment variable " +
			"UPCACHET_CACHET_APIKEY")
		return
	}

	client.Authentication.SetTokenAuth(config.CachetAPIkey)

	if config.UptimeRobotAPIkey == "" {
		log.Println("Please specify a Uptime Robot API key either through " +
			"the config file, or through environment variable " +
			"UPCACHET_UPTIMEROBOT_APIKEY")
		return
	}

	ur = urapi.New(config.UptimeRobotAPIkey)

	if config.BindAddress != "" {
		log.Println("Starting status verification server... ")
		// the point of this server is just to listen on a port and serve a
		// 200 OK so one can check whether the server is running with a browser
		// or e.g. Uptime Robot
		// it doesn't participate in the core functionality of Upcachet
		r := mux.NewRouter()
		r.HandleFunc("/", Index)
		http.Handle("/", r)
		go http.ListenAndServe(config.BindAddress, nil)
	}

	log.Println("Starting Uptime Robot monitor...")
	err = MonitorUptimeRobot()
	if err != nil {
		log.Println(err)
	} else {
		log.Println("Cleanly exiting")
	}
}
