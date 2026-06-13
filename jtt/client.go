package jtt

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"jtso/config"
	"jtso/logger"
	"net/http"
	"time"
)

// Client is the JTT REST API client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// State represents the response from GET /api/getstate
type State struct {
	Status  string `json:"status"`
	NumJobs int    `json:"num_jobs"`
}

// JobState represents the response from GET /api/jobstate
type JobState struct {
	JobID      string `json:"job_id"`
	Status     string `json:"status"`
	DeviceName string `json:"device_name"`
	Error      string `json:"error"`
}

// LeafResult represents a leaf-level test result
type LeafResult struct {
	GnmiLeaf           string   `json:"gnmi_leaf"`
	Description        string   `json:"description"`
	NetconfRpc         string   `json:"netconf_rpc"`
	NetconfLeaf        string   `json:"netconf_leaf"`
	CounterType        string   `json:"counter_type"`
	SpecificThresholds bool     `json:"specific_thresholds"`
	ValueRatio         int      `json:"value_ratio"`
	FalsePositive      int      `json:"false_positive"`
	TestType           int      `json:"test_type"`
	TestStatus         string   `json:"test_status"`
	TestDetail         []string `json:"test_detail"`
}

// PathResult represents a path-level test result
type PathResult struct {
	Subscription string       `json:"subscription"`
	Interval     int          `json:"interval"`
	Category     string       `json:"category"`
	Origin       string       `json:"origin"`
	Leaves       []LeafResult `json:"leaves"`
}

// JobResult represents the response from GET /api/jobresult
type JobResult struct {
	JobID       string       `json:"job_id"`
	Status      string       `json:"status"`
	DeviceName  string       `json:"device_name"`
	Version     string       `json:"version"`
	Error       string       `json:"error"`
	CompletedAt string       `json:"completed_at"`
	Model       string       `json:"model"`
	TestType    int          `json:"test_type"`
	ListOfPaths []PathResult `json:"listOfPaths"`
}

// ActiveJob represents an entry from GET /api/activejobs
type ActiveJob struct {
	JobID      string `json:"job_id"`
	Status     string `json:"status"`
	DeviceName string `json:"device_name"`
}

// GnmiCfg represents gNMI connection settings for a new job
type GnmiCfg struct {
	Port        int    `json:"port,omitempty"`
	Insecure    bool   `json:"insecure,omitempty"`
	SkipVerify  bool   `json:"skip_verify,omitempty"`
	ClientTls   bool   `json:"client_tls,omitempty"`
	User        string `json:"user,omitempty"`
	Pwd         string `json:"pwd,omitempty"`
	HideOrigin  bool   `json:"hide_origin,omitempty"`
	StreamMode  string `json:"stream_mode,omitempty"`
	MergeLeaves bool   `json:"merge_leaves,omitempty"`
}

// NetconfCfg represents NETCONF connection settings for a new job
type NetconfCfg struct {
	User    string `json:"user,omitempty"`
	Pwd     string `json:"pwd,omitempty"`
	Port    int    `json:"port,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

// LeafInput represents a leaf to test in a new job request
type LeafInput struct {
	GnmiLeaf           string `json:"gnmi_leaf"`
	Description        string `json:"description,omitempty"`
	NetconfRpc         string `json:"netconf_rpc,omitempty"`
	NetconfLeaf        string `json:"netconf_leaf,omitempty"`
	CounterType        string `json:"counter_type,omitempty"`
	SpecificThresholds bool   `json:"specific_thresholds,omitempty"`
	ValueRatio         int    `json:"value_ratio,omitempty"`
	FalsePositive      int    `json:"false_positive,omitempty"`
	TestType           int    `json:"test_type,omitempty"`
}

// XPathInput represents a subscription path in a new job request
type XPathInput struct {
	Subscription string      `json:"subscription"`
	Interval     int         `json:"interval"`
	Category     string      `json:"category,omitempty"`
	Origin       string      `json:"origin,omitempty"`
	Leaves       []LeafInput `json:"leaves"`
}

// NewJobRequest represents the POST /api/newjob request body
type NewJobRequest struct {
	RouterName          string       `json:"router_name"`
	Model               string       `json:"model,omitempty"`
	TestType            int          `json:"test_type,omitempty"`
	ForceSchemaDownload bool         `json:"force_schema_download,omitempty"`
	GnmiCfg             *GnmiCfg     `json:"gnmi_cfg,omitempty"`
	NetconfCfg          *NetconfCfg  `json:"netconf_cfg,omitempty"`
	XPaths              []XPathInput `json:"xpaths,omitempty"`
}

// NewJobResponse represents the POST /api/newjob response
type NewJobResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// CancelJobResponse represents the POST /api/canceljob response
type CancelJobResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// ErrorResponse represents a JTT error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// NewClient creates a new JTT API client from the JTT config
func NewClient(cfg *config.JTTConfig) *Client {
	transport := &http.Transport{}

	if cfg.UseSSL {
		cert, err := tls.LoadX509KeyPair(cfg.ClientCrt, cfg.ClientKey)
		if err != nil {
			logger.Log.Errorf("Error loading JTT client certificates: %v", err)
			transport.TLSClientConfig = &tls.Config{}
		} else {
			transport.TLSClientConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}
		}
	}

	return &Client{
		baseURL: cfg.URL,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}
}

// GetState calls GET /api/getstate
func (c *Client) GetState() (*State, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/getstate")
	if err != nil {
		logger.Log.Errorf("JTT GetState request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var state State
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		logger.Log.Errorf("JTT GetState decode failed: %v", err)
		return nil, err
	}
	return &state, nil
}

// GetJobState calls GET /api/jobstate?jobID=<id>
func (c *Client) GetJobState(jobID string) (*JobState, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/jobstate?jobID=" + jobID)
	if err != nil {
		logger.Log.Errorf("JTT GetJobState request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var js JobState
	if err := json.NewDecoder(resp.Body).Decode(&js); err != nil {
		logger.Log.Errorf("JTT GetJobState decode failed: %v", err)
		return nil, err
	}
	return &js, nil
}

// GetJobResult calls GET /api/jobresult?jobID=<id>
func (c *Client) GetJobResult(jobID string) (*JobResult, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/jobresult?jobID=" + jobID)
	if err != nil {
		logger.Log.Errorf("JTT GetJobResult request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var jr JobResult
	if err := json.NewDecoder(resp.Body).Decode(&jr); err != nil {
		logger.Log.Errorf("JTT GetJobResult decode failed: %v", err)
		return nil, err
	}
	return &jr, nil
}

// GetActiveJobs calls GET /api/activejobs
func (c *Client) GetActiveJobs() ([]ActiveJob, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/activejobs")
	if err != nil {
		logger.Log.Errorf("JTT GetActiveJobs request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var jobs []ActiveJob
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		logger.Log.Errorf("JTT GetActiveJobs decode failed: %v", err)
		return nil, err
	}
	return jobs, nil
}

// CancelJob calls POST /api/canceljob?jobID=<id>
func (c *Client) CancelJob(jobID string) (*CancelJobResponse, error) {
	resp, err := c.httpClient.Post(c.baseURL+"/api/canceljob?jobID="+jobID, "application/json", nil)
	if err != nil {
		logger.Log.Errorf("JTT CancelJob request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var cr CancelJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		logger.Log.Errorf("JTT CancelJob decode failed: %v", err)
		return nil, err
	}
	return &cr, nil
}

// NewJob calls POST /api/newjob
func (c *Client) NewJob(req *NewJobRequest) (*NewJobResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		logger.Log.Errorf("JTT NewJob marshal failed: %v", err)
		return nil, err
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/newjob", "application/json", bytes.NewReader(body))
	if err != nil {
		logger.Log.Errorf("JTT NewJob request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var nr NewJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&nr); err != nil {
		logger.Log.Errorf("JTT NewJob decode failed: %v", err)
		return nil, err
	}
	return &nr, nil
}

func (c *Client) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		return fmt.Errorf("JTT API error (%d): %s", resp.StatusCode, errResp.Error)
	}
	return fmt.Errorf("JTT API error (%d): %s", resp.StatusCode, string(body))
}
