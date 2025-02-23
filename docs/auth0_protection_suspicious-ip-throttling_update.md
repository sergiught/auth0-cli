---
layout: default
parent: auth0 protection suspicious-ip-throttling
has_toc: false
---
# auth0 protection suspicious-ip-throttling update

Update the suspicious ip throttling settings.

## Usage
```
auth0 protection suspicious-ip-throttling update [flags]
```

## Examples

```
  auth0 protection suspicious-ip-throttling update
  auth0 ap sit update --enabled true
  auth0 ap sit update --enabled true --allowlist "178.178.178.178"
  auth0 ap sit update --enabled true --allowlist "178.178.178.178" --shields block
  auth0 ap sit update -e true -l "178.178.178.178" -s block --json
```


## Flags

```
  -l, --allowlist strings           List of trusted IP addresses that will not have attack protection enforced against them. Comma-separated.
  -e, --enabled                     Enable (or disable) suspicious ip throttling.
      --json                        Output in json format.
      --pre-login-max int           Configuration options that apply before every login attempt. Total number of attempts allowed per day. (default 1)
      --pre-login-rate int          Configuration options that apply before every login attempt. Interval of time, given in milliseconds, at which new attempts are granted. (default 34560)
      --pre-registration-max int    Configuration options that apply before every user registration attempt. Total number of attempts allowed. (default 1)
      --pre-registration-rate int   Configuration options that apply before every user registration attempt. Interval of time, given in milliseconds, at which new attempts are granted. (default 1200)
  -s, --shields strings             Action to take when a suspicious IP throttling threshold is violated. Possible values: block, admin_notification. Comma-separated.
```


## Inherited Flags

```
      --debug           Enable debug mode.
      --no-color        Disable colors.
      --no-input        Disable interactivity.
      --tenant string   Specific tenant to use.
```


## Related Commands

- [auth0 protection suspicious-ip-throttling ips](auth0_protection_suspicious-ip-throttling_ips.md) - Manage blocked IP addresses
- [auth0 protection suspicious-ip-throttling show](auth0_protection_suspicious-ip-throttling_show.md) - Show suspicious ip throttling settings
- [auth0 protection suspicious-ip-throttling update](auth0_protection_suspicious-ip-throttling_update.md) - Update suspicious ip throttling settings


