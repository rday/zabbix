zabbix
======

This Go library implements the Zabbix 2.0 API. The Zabbix API is a JSONRPC
based API, although it is not compatable with Go's builtin JSONRPC libraries.
So we implement that JSONRPC, and provide data types that mimic Zabbbix's
return values.

Connecting to the API
=====================

```go
func main() {
    api, err := zabbix.NewAPI("http://zabbix.yourhost.net/api_jsonrpc.php", "User", "Password")
    if err != nil {
        fmt.Println(err)
        return
    }

    versionresult, err := api.Version()
    if err != nil {
        fmt.Println(err)
    }

    fmt.Println(versionresult)

    _, err = api.Login()
    if err != nil {
        fmt.Println(err)
        return
    }
    fmt.Println("Connected to API")
}
```

Making a call
=============

I typically wrap the actual API call to hide the messy details. If the
response has an Error field, and the code is greater than 0, the API
will return that Error. Then my wrapper function return a ZabbixError
to the caller.

```go
// Find and return a single host object by name
func GetHost(api *zabbix.API, host string) (zabbix.ZabbixHost, error) {
    params := make(map[string]interface{}, 0)
    filter := make(map[string]string, 0)
    filter["host"] = host
    params["filter"] = filter
    params["output"] = "extend"
    params["select_groups"] = "extend"
    params["templated_hosts"] = 1
    ret, err := api.Host("get", params)

    // This happens if there was an RPC error
    if err != nil {
        return nil, err
    }

    // If our call was successful
    if len(ret) > 0 {
        return ret[0], err
    }

    // This will be the case if the RPC call was successful, but
    // Zabbix had an issue with the data we passed.
    return nil, &zabbix.ZabbixError{0,"","Host not found"}
}
```
