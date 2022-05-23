package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/astarte-platform/astarte-go/client"
	"github.com/astarte-platform/astarte-go/interfaces"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Make sure SampleDatasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler, backend.StreamHandler interfaces. Plugin should not
// implement all these interfaces - only those which are required for a particular task.
// For example if plugin does not need streaming functionality then you are free to remove
// methods that implement backend.StreamHandler. Implementing instancemgmt.InstanceDisposer
// is useful to clean up resources used by previous datasource instance when a new datasource
// instance created upon datasource settings changed.
var (
	_ backend.QueryDataHandler    = (*AppEngineDatasource)(nil)
	_ backend.CheckHealthHandler  = (*AppEngineDatasource)(nil)
	_ backend.CallResourceHandler = (*AppEngineDatasource)(nil)
	// We're not interested in streaming
	// _ backend.StreamHandler         = (*SampleDatasource)(nil)
	_ instancemgmt.InstanceDisposer = (*AppEngineDatasource)(nil)
)

type appEngineDataSourceSourceSettings struct {
	ApiUrl string `json:"apiUrl"`
	Realm  string `json:"realm"`
	Token  string `json:"token"`
}

func newAppEngineDatasourceSettings(instanceSettings backend.DataSourceInstanceSettings) (appEngineDataSourceSourceSettings, error) {
	var settings appEngineDataSourceSourceSettings
	if err := json.Unmarshal(instanceSettings.JSONData, &settings); err != nil {
		return appEngineDataSourceSourceSettings{}, err
	}
	return settings, nil
}

// NewAppEngineDatasource creates a new datasource instance.
func NewAppEngineDatasource(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	log.DefaultLogger.Info("NewAppEngineDatasource called with", "backend_settings", settings)

	datasource := &AppEngineDatasource{}
	dsSettings, err := newAppEngineDatasourceSettings(settings)
	if err != nil {
		log.DefaultLogger.Error("Cannot read settings", "error", err)
		return nil, err
	}
	log.DefaultLogger.Info("Starting with settings:", "realm", dsSettings.Realm, "token", dsSettings.Token, "apiUrl", dsSettings.ApiUrl)

	// If localhost is used, one must specify AppEngine individual URL
	astarteAPIClient, err := client.NewClient(dsSettings.ApiUrl, nil)
	//astarteAPIClient, err := client.NewClientWithIndividualURLs(map[misc.AstarteService]string{misc.AppEngine: "http://localhost:4002"}, nil)
	if err != nil {
		log.DefaultLogger.Error("Cannot setup API client: ", "error", err)
		return nil, err
	}

	astarteAPIClient.SetToken(dsSettings.Token)
	datasource.astarteAPIClient = astarteAPIClient
	datasource.realm = dsSettings.Realm
	return datasource, nil
}

// AppEngineDatasource is a datasource which can respond to data queries and reports its health.
type AppEngineDatasource struct {
	astarteAPIClient *client.Client
	realm            string
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewAppEngineDatasource factory function.
func (d *AppEngineDatasource) Dispose() {
	// Delete the client (the one with AppEngine address and token)
	log.DefaultLogger.Info("Disposing of", "appengine_datasource", d)
	// Clean up datasource instance resources.
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *AppEngineDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	log.DefaultLogger.Info("QueryData called", "request", req)

	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := d.query(ctx, req.PluginContext, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	log.DefaultLogger.Info("Returning response", "response", response)
	return response, nil
}

type queryModel struct {
	Device        string `json:"device"`
	InterfaceName string `json:"interfaceName"`
	Path          string `json:"path"`
}

func (d *AppEngineDatasource) query(_ context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	response := backend.DataResponse{}

	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	log.DefaultLogger.Info("Received query JSON", "json_as_string", string(query.JSON))

	response.Error = json.Unmarshal(query.JSON, &qm)
	if response.Error != nil {
		log.DefaultLogger.Error("Error in query model unmarshal", "error", response.Error)
		return response
	}

	// create data frame response.
	frame := data.NewFrame("response")

	paginator, err := d.astarteAPIClient.AppEngine.GetDatastreamsTimeWindowPaginator(d.realm, qm.Device, client.AstarteDeviceID, qm.InterfaceName,
		qm.Path, query.TimeRange.From, query.TimeRange.To, client.AscendingOrder)

	if err != nil {
		response.Error = err
		return response
	}

	timestamps := []time.Time{}
	values := []float64{}

	for ok := true; ok; ok = paginator.HasNextPage() {
		page, err := paginator.GetNextPage()
		if err != nil {
			log.DefaultLogger.Error("Next page paginator error", "error", err)
			response.Error = err
			return response
		}

		log.DefaultLogger.Info("Start reading Astarte data")

		for _, v := range page {
			switch v.Value.(type) {
			case float64:
				timestamps = append(timestamps, v.Timestamp)
				values = append(values, v.Value.(float64))
			case int64:
				timestamps = append(timestamps, v.Timestamp)
				values = append(values, float64(v.Value.(int64)))
			case string:
				if f, err := strconv.ParseFloat(v.Value.(string), 64); err != nil {
					log.DefaultLogger.Warn("Could not parse as numeric datatype", "value", v.Value, "error", err)
				} else {
					timestamps = append(timestamps, v.Timestamp)
					values = append(values, f)
				}
			default:
				response.Error = fmt.Errorf("Device %s has data of non-numeric type on interface %s, path %s", qm.Device, qm.InterfaceName, qm.Path)
				log.DefaultLogger.Error("Error on value type read", "error", response.Error)
				return response
			}
		}
	}

	log.DefaultLogger.Info("Successful Astarte data reading")

	TimeField := data.NewField("Time", nil, timestamps)
	log.DefaultLogger.Info("Successful time field creation")

	ValueField := data.NewField("Value", nil, values)
	log.DefaultLogger.Info("Successful value field creation")

	frame.Fields = append(frame.Fields, TimeField, ValueField)
	log.DefaultLogger.Info("Successful frame field append")

	// add the frames to the response.
	response.Frames = append(response.Frames, frame)
	log.DefaultLogger.Info("Successful response frame append", "response", response)

	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *AppEngineDatasource) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	log.DefaultLogger.Info("CheckHealth called", "request", req)

	var status = backend.HealthStatusOk
	var message = "Data source is working"

	// Run an actual query to Astarte, so that our JWT is checked, too
	_, err := d.astarteAPIClient.AppEngine.GetDevicesStats(d.realm)

	if err != nil {
		log.DefaultLogger.Error("CheckHealth error", "err", err)
		status = backend.HealthStatusError
		message = err.Error()
	}

	return &backend.CheckHealthResult{
		Status:  status,
		Message: message,
	}, nil
}

func (d *AppEngineDatasource) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	log.DefaultLogger.Info("CallResource  called", "request", req)

	u, _ := url.Parse(req.URL)
	params, _ := url.ParseQuery(u.RawQuery)

	if params["device_id"] != nil {
		// if device_id is provided, we've been asked for device introspection
		interfaces, err := d.getDeviceIntrospection(params["device_id"][0])
		if err != nil {
			sendBadRequest(err, sender)
		}
		body, _ := json.Marshal(interfaces)
		return sendResult(body, sender)
	} else if params["name"] != nil && params["major"] != nil {
		// we assume a valid int is always passed as interface major value
		major, _ := strconv.Atoi(params["major"][0])
		iface, err := d.getInterface(params["name"][0], major)
		if err != nil {
			sendBadRequest(err, sender)
		}
		body, _ := json.Marshal(iface)
		return sendResult(body, sender)
	} else {
		// don't know what else could we provide
		return sendBadRequest(fmt.Errorf("unexpected request"), sender)
	}
}

func (d *AppEngineDatasource) getInterface(interfaceName string, interfaceMajor int) (interfaces.AstarteInterface, error) {
	interfaceDoc, err := d.astarteAPIClient.RealmManagement.GetInterface(d.realm, interfaceName, interfaceMajor)
	if err != nil {
		log.DefaultLogger.Error("Can't get interface data", "err", err, "interface", interfaceName, "interfaceMajor", interfaceMajor)
		return interfaces.AstarteInterface{}, err
	}
	log.DefaultLogger.Info("Received doc for interface", "interface", interfaceName, "major", interfaceMajor)

	return interfaceDoc, nil
}

type introspectionEntry struct {
	Name  string `json:"name"`
	Major int    `json:"major"`
	Minor int    `json:"minor"`
}

func (d *AppEngineDatasource) getDeviceIntrospection(deviceID string) ([]introspectionEntry, error) {
	details, err := d.astarteAPIClient.AppEngine.GetDevice(d.realm, deviceID, client.AstarteDeviceID)
	if err != nil {
		log.DefaultLogger.Error("Can't get device introspection", "err", err, "device_id", deviceID)
		return nil, err
	}
	log.DefaultLogger.Info("Received Astarte introspection for device", "device_id", deviceID)
	interfaces := []introspectionEntry{}
	for interfaceName, interfaceDetails := range details.Introspection {
		interfaces = append(interfaces, introspectionEntry{Name: interfaceName, Major: interfaceDetails.Major, Minor: interfaceDetails.Minor})
	}

	return interfaces, nil
}

func sendResult(body []byte, sender backend.CallResourceResponseSender) error {
	log.DefaultLogger.Info("Sending call resource response")
	return sender.Send(&backend.CallResourceResponse{
		Status: http.StatusOK,
		Body:   body,
	})
}

func sendBadRequest(err error, sender backend.CallResourceResponseSender) error {
	return sender.Send(&backend.CallResourceResponse{
		Status: http.StatusBadRequest,
		Body:   []byte(err.Error()),
	})
}
