package zabbix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

/**
Zabbix and Go's RPC implementations don't play with each other.. at all.
So I've re-created the wheel at bit.
*/
type JsonRPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Error   ZabbixError `json:"error"`
	Result  interface{} `json:"result"`
	Id      int         `json:"id"`
}

type JsonRPCRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`

	// Zabbix 2.0:
	// The "user.login" method must be called without the "auth" parameter
	Auth string `json:"auth,omitempty"`
	Id   int    `json:"id"`
}

type ZabbixError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

func (z *ZabbixError) Error() string {
	return z.Data
}

type ZabbixHost map[string]interface{}
type ZabbixGraph map[string]interface{}
type ZabbixGraphItem map[string]interface{}
type ZabbixHistoryItem struct {
	Clock  string `json:"clock"`
	Value  string `json:"value"`
	Itemid string `json:"itemid"`
}

type API struct {
	url    string
	user   string
	passwd string
	id     int
	auth   string
	Client *http.Client
}

func NewAPI(server, user, passwd string) (*API, error) {
	return &API{server, user, passwd, 0, "", &http.Client{}}, nil
}

func (api *API) GetAuth() string {
	return api.auth
}

/**
Each request establishes its own connection to the server. This makes it easy
to keep request/responses in order without doing any concurrency
*/

func (api *API) ZabbixRequest(method string, data interface{}) (JsonRPCResponse, error) {
	// Setup our JSONRPC Request data
	id := api.id
	api.id = api.id + 1
	jsonobj := JsonRPCRequest{"2.0", method, data, api.auth, id}
	encoded, err := json.Marshal(jsonobj)

	if err != nil {
		return JsonRPCResponse{}, err
	}

	// Setup our HTTP request
	request, err := http.NewRequest("POST", api.url, bytes.NewBuffer(encoded))
	if err != nil {
		return JsonRPCResponse{}, err
	}
	request.Header.Add("Content-Type", "application/json-rpc")
	if api.auth != "" {
		// XXX Not required in practice, check spec
		//request.SetBasicAuth(api.user, api.passwd)
		//request.Header.Add("Authorization", api.auth)
	}

	// Execute the request
	response, err := api.Client.Do(request)
	if err != nil {
		return JsonRPCResponse{}, err
	}

	/**
	We can't rely on response.ContentLength because it will
	be set at -1 for large responses that are chunked. So
	we treat each API response as streamed data.
	*/
	var result JsonRPCResponse
	var buf bytes.Buffer

	_, err = io.Copy(&buf, response.Body)
	if err != nil {
		return JsonRPCResponse{}, err
	}

	json.Unmarshal(buf.Bytes(), &result)

	response.Body.Close()

	return result, nil
}

func (api *API) Login() (bool, error) {
	params := make(map[string]string, 0)
	params["user"] = api.user
	params["password"] = api.passwd

	response, err := api.ZabbixRequest("user.login", params)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return false, err
	}

	if response.Error.Code != 0 {
		return false, &response.Error
	}

	api.auth = response.Result.(string)
	return true, nil
}

func (api *API) Logout() (bool, error) {
	emptyparams := make(map[string]string, 0)
	response, err := api.ZabbixRequest("user.logout", emptyparams)
	if err != nil {
		return false, err
	}

	if response.Error.Code != 0 {
		return false, &response.Error
	}

	return true, nil
}

func (api *API) Version() (string, error) {
	response, err := api.ZabbixRequest("APIInfo.version", make(map[string]string, 0))
	if err != nil {
		return "", err
	}

	if response.Error.Code != 0 {
		return "", &response.Error
	}

	return response.Result.(string), nil
}

/**
Interface to the user.* calls
*/
func (api *API) User(method string, data interface{}) ([]interface{}, error) {
	response, err := api.ZabbixRequest("user."+method, data)
	if err != nil {
		return nil, err
	}

	if response.Error.Code != 0 {
		return nil, &response.Error
	}

	return response.Result.([]interface{}), nil
}

/**
Interface to the host.* calls
*/
func (api *API) Host(method string, data interface{}) ([]ZabbixHost, error) {
	response, err := api.ZabbixRequest("host."+method, data)
	if err != nil {
		return nil, err
	}

	if response.Error.Code != 0 {
		return nil, &response.Error
	}

	// XXX uhg... there has got to be a better way to convert the response
	// to the type I want to return
	res, err := json.Marshal(response.Result)
	var ret []ZabbixHost
	err = json.Unmarshal(res, &ret)
	return ret, nil
}

/**
Interface to the graph.* calls
*/
func (api *API) Graph(method string, data interface{}) ([]ZabbixGraph, error) {
	response, err := api.ZabbixRequest("graph."+method, data)
	if err != nil {
		return nil, err
	}

	if response.Error.Code != 0 {
		return nil, &response.Error
	}

	// XXX uhg... there has got to be a better way to convert the response
	// to the type I want to return
	res, err := json.Marshal(response.Result)
	var ret []ZabbixGraph
	err = json.Unmarshal(res, &ret)
	return ret, nil
}

/**
Interface to the history.* calls
*/
func (api *API) History(method string, data interface{}) ([]ZabbixHistoryItem, error) {
	response, err := api.ZabbixRequest("history."+method, data)
	if err != nil {
		return nil, err
	}

	if response.Error.Code != 0 {
		return nil, &response.Error
	}

	// XXX uhg... there has got to be a better way to convert the response
	// to the type I want to return
	res, err := json.Marshal(response.Result)
	var ret []ZabbixHistoryItem
	err = json.Unmarshal(res, &ret)
	return ret, nil
}
