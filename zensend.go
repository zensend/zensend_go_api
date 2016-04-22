package zensend

import (
	"encoding/json"
	"math"
	"net/http"
	"net/url"
	"strings"
)

const (
	URL = "https://api.zensend.io"
	VERIFY_URL = "https://verify.zensend.io"
)

type Client struct {
	APIKey     string
	HTTPClient *http.Client
	URL        string
	VerifyURL  string
}

func New(APIKey string) *Client {
	return NewWithURL(APIKey, URL)
}

func NewWithURL(APIKey string, URL string) *Client {
	return NewWithURLAndVerifyURL(APIKey, URL, VERIFY_URL)
	
}

func NewWithURLAndVerifyURL(APIKey string, URL string, VerifyURL string) *Client {
	return &Client{APIKey: APIKey, HTTPClient: &http.Client{}, URL: URL, VerifyURL: VerifyURL}
}

type zenSendSendSMSResponse struct {
	Success *SendSMSResponse
	Failure *failure
}

type zenSendOperatorLookupResponse struct {
	Success *OperatorLookupResponse
	Failure *failure
}

type zenSendCheckBalanceResponse struct {
	Success *checkBalanceResponse
	Failure *failure
}

type zenSendGetPricesResponse struct {
	Success *getPricesResponse
	Failure *failure
}

type zenSendCreateMsisdnVerificationResponse struct {
	Success *createMsisdnVerificationResponse
	Failure *failure
}

type zenSendMsisdnVerificationStatusResponse struct {
	Success *msisdnVerificationStatusResponse
	Failure *failure
}

type getPricesResponse struct {
	PricesInPence map[string]float64 `json:"prices_in_pence"`
}

type checkBalanceResponse struct {
	Balance float64
}

type createMsisdnVerificationResponse struct {
	Session string
}

type msisdnVerificationStatusResponse struct {
	Msisdn string
}

type failure struct {
	FailCode  string
	Parameter string

	CostInPence       *float64 `json:"cost_in_pence"`
	NewBalanceInPence *float64 `json:"new_balance_in_pence"`
}

func (c *Client) MsisdnVerificationStatus(session string) (string, error) {
	msisdnVerificationStatusResponse := &zenSendMsisdnVerificationStatusResponse{};

	httpStatus, requestError := c.makeRequest(msisdnVerificationStatusResponse, c.VerifyURL + "/api/msisdn_verify?SESSION=" + url.QueryEscape(session), nil)

	if requestError != nil {
		return "", requestError
	}

	if msisdnVerificationStatusResponse.Success != nil {
		return msisdnVerificationStatusResponse.Success.Msisdn, nil
	}

	return "", createError(msisdnVerificationStatusResponse.Failure, httpStatus)

}

func (c *Client) CreateMsisdnVerification(number string) (string, error) {
	return c.CreateMsisdnVerificationWithOptions(number, VerifyOptions{})
}

func (c *Client) CreateMsisdnVerificationWithOptions(number string, options VerifyOptions) (string, error) {
	createMsisdnVerificationResponse := &zenSendCreateMsisdnVerificationResponse{};

	postParams := url.Values{}
	postParams.Set("NUMBER", number)
	if len(options.Originator) > 0 {
		postParams.Set("ORIGINATOR", options.Originator)
	}

	if len(options.Message) > 0 {
		postParams.Set("MESSAGE", options.Message)
	}

	httpStatus, requestError := c.makeRequest(createMsisdnVerificationResponse, c.VerifyURL + "/api/msisdn_verify", postParams)

	if requestError != nil {
		return "", requestError
	}

	if createMsisdnVerificationResponse.Success != nil {
		return createMsisdnVerificationResponse.Success.Session, nil
	}

	return "", createError(createMsisdnVerificationResponse.Failure, httpStatus)

}

func (c *Client) CheckBalance() (float64, error) {
	checkBalanceResponse := &zenSendCheckBalanceResponse{}

	httpStatus, requestError := c.makeRequest(checkBalanceResponse, c.URL+"/v3/checkbalance", nil)

	if requestError != nil {
		return math.NaN(), requestError
	}

	if checkBalanceResponse.Success != nil {
		return checkBalanceResponse.Success.Balance, nil
	}
	return math.NaN(), createError(checkBalanceResponse.Failure, httpStatus)

}

func (c *Client) GetPrices() (map[string]float64, error) {
	getPricesResponse := &zenSendGetPricesResponse{}

	httpStatus, requestError := c.makeRequest(getPricesResponse, c.URL+"/v3/prices", nil)

	if requestError != nil {
		return nil, requestError
	}

	if getPricesResponse.Success != nil {
		return getPricesResponse.Success.PricesInPence, nil
	}
	return nil, createError(getPricesResponse.Failure, httpStatus)

}

func (c *Client) LookupOperator(number string) (*OperatorLookupResponse, error) {

	operatorLookupResponse := &zenSendOperatorLookupResponse{}

	httpStatus, requestError := c.makeRequest(operatorLookupResponse, c.URL+"/v3/operator_lookup?NUMBER="+url.QueryEscape(number), nil)

	if requestError != nil {
		return nil, requestError
	}

	if operatorLookupResponse.Success != nil {
		return operatorLookupResponse.Success, nil
	}
	return nil, createError(operatorLookupResponse.Failure, httpStatus)
}

func (c *Client) SendSMS(message *Message) (*SendSMSResponse, error) {
	sendSmsResponse := &zenSendSendSMSResponse{}

	postParams, error := message.toPostParams()

	if error != nil {
		return sendSmsResponse.Success, error
	}

	httpStatus, requestError := c.makeRequest(sendSmsResponse, c.URL+"/v3/sendsms", postParams)

	if requestError != nil {
		return nil, requestError
	}

	if sendSmsResponse.Success != nil {
		return sendSmsResponse.Success, nil
	}
	return nil, createError(sendSmsResponse.Failure, httpStatus)

}

func createError(failure *failure, StatusCode int) ZenSendError {
	if failure == nil {
		return ZenSendError{StatusCode: StatusCode}
	}
	return ZenSendError{
		StatusCode:        StatusCode,
		FailCode:          failure.FailCode,
		Parameter:         failure.Parameter,
		CostInPence:       failure.CostInPence,
		NewBalanceInPence: failure.NewBalanceInPence}
}

func (c *Client) makeRequest(responseObject interface{}, fullPath string, params url.Values) (int, error) {
	uri, error := url.Parse(fullPath)

	if error != nil {
		return -1, error
	}

	var request *http.Request

	if params != nil {
		body := params.Encode()

		if request, error = http.NewRequest("POST", uri.String(), strings.NewReader(body)); error != nil {
			return -1, error
		}

		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {

		if request, error = http.NewRequest("GET", uri.String(), nil); error != nil {
			return -1, error
		}
	}

	request.Header.Add("X-API-KEY", c.APIKey)

	httpResponse, error := c.HTTPClient.Do(request)

	if httpResponse != nil && httpResponse.Body != nil {
		defer httpResponse.Body.Close()
	}

	if error != nil {
		return -1, error
	}

	if strings.Contains(httpResponse.Header.Get("Content-Type"), "application/json") {
		decoder := json.NewDecoder(httpResponse.Body)

		if error = decoder.Decode(&responseObject); error != nil {
			return httpResponse.StatusCode, error
		}
		return httpResponse.StatusCode, nil
	}
	return httpResponse.StatusCode, ZenSendError{StatusCode: httpResponse.StatusCode, FailCode: "", Parameter: ""}
}
