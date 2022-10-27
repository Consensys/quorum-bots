package http

import (
	"io"
	"log"
	"net/http"

	"upgradebot/config"
)

type HTTPClient struct {
	httpClient *http.Client
	config     *config.Config
}

func newHttpAdapter(config *config.Config) *HTTPClient {
	client := &http.Client{}
	return &HTTPClient{
		httpClient: client,
		config:     config,
	}
}

func (adapter *HTTPClient) DoPost(url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	resp, err := adapter.do(req)
	if err != nil {
		return nil, err
	}
	return adapter.deserialize(resp)
}

func (adapter *HTTPClient) DoGet(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := adapter.do(req)
	if err != nil {
		return nil, err
	}
	return adapter.deserialize(resp)
}

func (adapter *HTTPClient) deserialize(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (adapter *HTTPClient) do(req *http.Request) (*http.Response, error) {
	log.Printf("%s %s\n", req.Method, req.URL)
	req.SetBasicAuth(adapter.config.GithubUsername, adapter.config.GithubUserToken)
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	resp, err := adapter.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	// For DEBUG. print HTTP headers
	//for name, values := range resp.Header {
	//	// Loop over all values for the name.
	//	for _, value := range values {
	//		fmt.Println(name, value)
	//	}
	//}
	return resp, nil

}
