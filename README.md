# Upcachet
![Upcachet logo](https://github.com/tnyim/upcachet/raw/master/upcachet.png)
This is a small tool (microservice, if you wish) that automatically updates a [Cachet](https://cachethq.io/)-powered status page based on the status of monitors in an [Uptime Robot](https://uptimerobot.com) account. It can update Cachet component status and Cachet metrics (from the response times measured by Uptime Robot).

It is meant to run continuously in a server, possibly the same server where Cachet is installed.

It is used in production to update the http://status.tny.im/ status page.

# Getting started

#### If you already have a Go development environment

Simply get the package and its dependencies,

`go get -u github.com/tnyim/upcachet`

...and use your favorite method (e.g. `go build` in the package root) to get a binary.

#### If you don't have a Go development environment

You can simply clone this repo and use [Hellogopher](https://github.com/cloudflare/hellogopher) to `make` the project.

# Configuration

Upcachet needs to be configured before it can be useful.

By default (if no command line arguments are passed), Upcachet uses `config.json` (in the same directory as the binary) as the configuration file.

You can specify a custom path to the config file like this:

`./upcachet -c path/to/my/config/file.json`

**The first time you run Upcachet, it will create a skeleton, almost empty, config file.** You will want to begin by configuring the `CachetAPIkey`, `CachetEndpoint` and `UptimeRobotAPIkey` fields:

- Get the Cachet API key from the user profile page of your Cachet install (the page at `/dashboard/user`)
- The Cachet endpoint is the URL to your Cachet install. For example, in our case it is `http://status.tny.im/`
- Get the Uptime Robot API key from your [account settings (scroll down)](https://uptimerobot.com/dashboard.php#mySettings).

The API keys and Cachet endpoint can also be specified through environment variables: `UPCACHET_CACHET_APIKEY`, `UPCACHET_CACHET_ENDPOINT` and `UPCACHET_UPTIMEROBOT_APIKEY`.

For the environment vars to work, ensure that the respective fields in the config file are completely missing (i.e. not even their keys are there). Otherwise, they will take precedence (even if empty).

**You are now ready to run Upcachet for the second time.** It should now be able to access your Uptime Robot and Cachet accounts and will print a list of monitors for the former and components for the latter. It will also add some example config entries to the config file, and exit.

The next step is to configure which monitors affect which Cachet components, using the monitor and component IDs dumped by Cachet. This is what Upcachet calls `MonitorComponentMap`. As the name indicates, this is a map/dictionary that maps Uptime Robot monitors to **lists** of Cachet components to update when that monitor goes up or down.

This way, a single monitor can affect the status of multiple components. For example, if you have a server hosting multiple services, you'll want the components for all of those services to change when the server goes down.

`MonitorMetricMap` works in a similar way, except it maps to Cachet metrics instead. If you don't care about response time metrics, you can leave this empty.

```
    "MonitorComponentMap": {
        "776260090": [    <----- put Uptime Robot monitor IDs here
            4, 7          <----- put Cachet component IDs here
        ],
        "775543690": [
            8, 10, 11, 6
        ],
        "778180514": [
            11
        ]
    },
    "MonitorMetricMap": {
        "777244008": [    <----- put Uptime Robot monitor IDs here
            1             <----- put Cachet metric IDs here
        ],
        "778180514": [
            2
        ]
    },

```

## Other configuration options

`CheckInterval` allows for specifying how often, in seconds, Upcachet should update Uptime Robot monitor status and response times. This value is in **nanoseconds**. If zero (default), Upcachet will use a check interval that is half of the interval used by Uptime Robot between pings (for free accounts, this is five minutes).

`BindAddress`: if not empty (default), Upcachet will start a HTTP server bound to the value of this setting (for example, `localhost:8500`). The only purpose of this server is to be able to check whether Upcachet is running or not (so you have something you can point e.g. Uptime Robot at).

# FAQ
## Why not use the webhook contact option provided by Uptime Robot?

We tried this for some months and the results were not satisfactory. Sometimes the webhooks were not being called on time if at all, and configuring different webhook contact alerts for each monitor (so we could know which component) was a very time-consuming task.

After that initial trial, and since we'd always need to have some sort of custom tool running (the Uptime Robot webhooks are not directly compatible with anything Cachet accepts) we decided it would be best to use an active monitoring approach, where the status of all Uptime Robot monitors is checked often enough and the Cachet components updated in reposnse to monitor status changes.

This has the added advantage of being able to update Cachet metrics with the monitor response time.
