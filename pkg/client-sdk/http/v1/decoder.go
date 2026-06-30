package v1

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"

	"github.com/marmotedu/component-base/pkg/core"
	"github.com/marmotedu/errors"
)

func DecodeResponse[RESP any](ctx context.Context, response *http.Response) (interface{}, error) {
	defer response.Body.Close()
	bodydata, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read response body")
	}

	if len(bodydata) == 0 || bodydata == nil {
		if response.StatusCode != http.StatusOK {
			return nil, errors.New(response.Status)
		}
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		errResp := new(core.ErrResponse)
		err = json.Unmarshal(bodydata, errResp)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal error response, status: %s, response raw: %s", response.Status, string(bodydata))
		}
		return nil, errors.WithCode(errResp.Code, errResp.Message)
	}

	// Handle JSON null value for NilBody type (server returns "null" for empty response)
	if isNilBody[RESP]() && string(bodydata) == "null" {
		return nil, nil
	}

	// Check if RESP is NilBody type but response body is not empty
	if isNilBody[RESP]() {
		return nil, errors.New("response declared as NilBody type but actual response body is not empty")
	}

	// Get the type of RESP
	respType := reflect.TypeOf((*RESP)(nil)).Elem()

	// If RESP is a pointer type, we can unmarshal directly to a new instance of it
	// If RESP is a non-pointer type, we need to unmarshal to the address of a new instance
	var result interface{}

	if respType.Kind() == reflect.Ptr {
		// RESP is a pointer type like *T
		// Create a new pointer to the underlying type
		ptrValue := reflect.New(respType.Elem())
		err = json.Unmarshal(bodydata, ptrValue.Interface())
		if err != nil {
			return nil, err
		}
		// Get the pointer value that contains our data
		result = ptrValue.Interface()
	} else {
		// RESP is a non-pointer type like T
		// Create a new instance and unmarshal to its address
		newValue := reflect.New(respType)
		err = json.Unmarshal(bodydata, newValue.Interface())
		if err != nil {
			return nil, err
		}
		// Dereference to get the actual value
		result = newValue.Elem().Interface()
	}

	r, ok := result.(RESP)
	if !ok {
		return nil, errors.Errorf("failed to convert interface to %s", respType.String())
	}
	return r, err
}
