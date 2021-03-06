package mon

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	"github.com/go-resty/resty/v2"
)

type IHasura interface {
	// SetResponseKey sets response key
	// e.g. in hasura return data.user then response key is "user"
	SetResponseKey(key string) IHasura

	// SetResponseModel sets response model/schema
	// Struct with json tag
	SetResponseModel(model interface{}) IHasura

	// Exec returns nil if calling graphql API returns success
	Exec(variables interface{}, headers map[string]string) error
}

type Hasura struct {
	GraphqlURL string
	Secret     string
	Query      string

	ResponseKey   string
	ResponseModel interface{}
}

func (h *Hasura) SetResponseKey(key string) IHasura {
	h.ResponseKey = key
	return h
}

func (h *Hasura) SetResponseModel(model interface{}) IHasura {
	h.ResponseModel = model
	return h
}

func (h *Hasura) Exec(variables interface{}, headers map[string]string) error {
	body := make(map[string]interface{})
	body["query"] = h.Query
	body["variables"] = variables

	jsonbody, err := json.Marshal(&body)
	if err != nil {
		return err
	}

	client := resty.New()
	req := client.R().SetHeader("Content-Type", "application/json")
	if h.Secret != "" {
		req = req.SetHeader("x-hasura-admin-secret", h.Secret)
	}
	for key, value := range headers {
		req = req.SetHeader(key, value)
	}

	resp, err := req.SetBody(jsonbody).Post(h.GraphqlURL)
	if err != nil {
		return err
	}

	response := new(HasuraResponseSchema)
	err = json.Unmarshal(resp.Body(), response)
	if err != nil {
		return err
	}

	if response.Errors != nil {
		return errors.New("something went wrong when calling hasura")
	}

	values, ok := response.Data.(map[string]interface{})[h.ResponseKey]
	if !ok {
		return errors.New("failed to parse response from hasura")
	}

	b, err := json.Marshal(values)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, h.ResponseModel)
}

type HasuraResponseSchema struct {
	Data   interface{} `json:"data,omitempty"`
	Errors interface{} `json:"errors,omitempty"`
}

// NewHasura initiates hasura client object
// @params filepath set the .gql file
func NewHasura(graphqlURL string, secret string, filepath string) (IHasura, error) {
	h := &Hasura{
		GraphqlURL: graphqlURL,
		Secret:     secret,
	}

	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	h.Query = string(file)

	return h, nil
}
