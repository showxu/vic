package events

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"io"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/swag"

	"github.com/go-openapi/strfmt"

	"github.com/go-swagger/go-swagger/fixtures/bugs/84/models"
)

// NewPutEventByIDParams creates a new PutEventByIDParams object
// with the default values initialized.
func NewPutEventByIDParams() PutEventByIDParams {
	var ()
	return PutEventByIDParams{}
}

// PutEventByIDParams contains all the bound params for the put event by Id operation
// typically these are obtained from a http.Request
//
// swagger:parameters putEventById
type PutEventByIDParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request

	/*Existing event
	  Required: true
	  In: body
	*/
	Event *models.Event
	/*Existing event id.
	  Required: true
	  In: path
	*/
	ID int64
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls
func (o *PutEventByIDParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error
	o.HTTPRequest = r

	if runtime.HasBody(r) {
		defer r.Body.Close()
		var body models.Event
		if err := route.Consumer.Consume(r.Body, &body); err != nil {
			if err == io.EOF {
				res = append(res, errors.Required("event", "body"))
			} else {
				res = append(res, errors.NewParseError("event", "body", "", err))
			}

		} else {
			if err := body.Validate(route.Formats); err != nil {
				res = append(res, err)
			}

			if len(res) == 0 {
				o.Event = &body
			}
		}

	} else {
		res = append(res, errors.Required("event", "body"))
	}

	rID, rhkID, _ := route.Params.GetOK("id")
	if err := o.bindID(rID, rhkID, route.Formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *PutEventByIDParams) bindID(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	value, err := swag.ConvertInt64(raw)
	if err != nil {
		return errors.InvalidType("id", "path", "int64", raw)
	}
	o.ID = value

	return nil
}