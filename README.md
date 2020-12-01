# Blackbox Exporter - ScienceMesh edition

This is a fork of the [Blackbox Exporter](https://github.com/prometheus/blackbox_exporter) extended for the ScienceMesh project.

## Documentation
The original documentation of the Blackbox Exporter can be found [here](https://github.com/prometheus/blackbox_exporter/README.md).

### The Nagios prober
The main extension of this fork is the addition of a Nagios prober which can be used to perform  checks using any Nagios probe. It can be configured as follows:
```
modules:
  nagios_test:
    prober: nagios
    nagios:
      check: check_test
      args:
        - --host $target$
        - --something=42
      proxy_url: 'http://someproxy:80'
      treat_warnings_as_failure: true
```

The following values are supported for the `nagios` prober:

| Value | Description | Default |
| --- | --- | --- | 
| `check` | The check name; the corresponding binary must either reside in the `checks` subdirectory or anywhere on the system's `PATH` environment variable. |
| `args` | An array of command-line arguments passed to the Nagios probe; see below for placeholders support. | `[]` |
| `proxy_url` | An optional proxy URL to use; note that this only sets corresponding environment variables internally (`HTTP_PROXY`, `HTTPS_PROXY`, etc.), so the Nagios probe needs to utilize these. 
| `treat_warnings_as_failure` | The Blackbox Exporter reports whether a check succeeded in the `probe_succes` metric; if this setting is set to `true`, a _Warning_ result will be treated as an unsuccessful check. | `false` |
 
#### Argument placeholders
Placeholders in arguments passed to the Nagios probe via the `args` value have the form `$name$`:
```
args:
  - --host $target$ 
```

The following standard placeholders are supported:

| Placeholder | Description |
| --- | --- |
| `target` | The host name that should be used as the Nagios probe's target. |

Any additional parameters passed by URL can also be used as placeholders:
```
http://blackbox.example.com/probe?module=nagios_test&target=google.com&my_param=123
```
The custom parameter `my_param` can then be accessed as `$my_param$` in the probe arguments.

Also note that placeholder names are _not_ case-sensitive.

### A note on building
The included `Makefile` should only be used for building inside a Docker container.
