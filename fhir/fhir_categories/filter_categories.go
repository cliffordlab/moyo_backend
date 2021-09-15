package fhir_categories

import (
	"encoding/json"
)

func rawMarshal(v interface{}) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

type Root struct {
	fields struct {
		Bundle *Bundle `json:"Bundle,omitempty"`
	}
	other map[string]interface{}
}

func (v *Root) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &v.fields); err != nil {
		return err
	}
	if v.other == nil {
		v.other = make(map[string]interface{})
	}
	if err := json.Unmarshal(b, &v.other); err != nil {
		return err
	}
	return nil
}

func (v *Root) MarshalJSON() ([]byte, error) {
	var err error
	if v.other["Bundle"], err = rawMarshal(v.fields.Bundle); err != nil {
		return nil, err
	}
	return json.Marshal(v.other)
}

type Bundle struct {
	fields struct {
		Entry []Entry `json:"entry,omitempty"`
	}
	other map[string]interface{}
}

func (v *Bundle) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &v.fields); err != nil {
		return err
	}
	if v.other == nil {
		v.other = make(map[string]interface{})
	}
	if err := json.Unmarshal(b, &v.other); err != nil {
		return err
	}
	return nil
}

func (v *Bundle) MarshalJSON() ([]byte, error) {
	var err error
	if v.other["entry"], err = rawMarshal(v.fields.Entry); err != nil {
		return nil, err
	}
	return json.Marshal(v.other)
}

type Entry struct {
	fields struct {
		Resource *Resource `json:"resource,omitempty"`
	}
	other map[string]interface{}
}

func (v *Entry) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &v.fields); err != nil {
		return err
	}
	if v.other == nil {
		v.other = make(map[string]interface{})
	}
	if err := json.Unmarshal(b, &v.other); err != nil {
		return err
	}
	return nil
}

func (v *Entry) MarshalJSON() ([]byte, error) {

	var err error
	if v.other["resource"], err = rawMarshal(v.fields.Resource); err != nil {
		return nil, err
	}
	return json.Marshal(v.other)
}

type Resource struct {
	fields struct {
		//Patient *Patient `json:"patient,omitempty"`
		Condition *Condition `json:"Condition,omitempty"`
		Allergy *Allergy `json:"AllergyIntolerance,omitempty"`
		CarePlan *CarePlan `json:"CarePlan,omitempty"`
		Document *Document `json:"DocumentReference,omitempty"`
		Immunization *Immunization `json:"Immunization,omitempty"`
		Medication *Medication `json:"MedicationOrder,omitempty"`
		Observation *Observation `json:"Observation,omitempty"`
		Procedure *Procedure `json:"Procedure,omitempty"`
		Report *Report `json:"DiagnosticReport,omitempty"`
	}
	other map[string]interface{}
}

func (v *Resource) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &v.fields); err != nil {
		return err
	}
	if v.other == nil {
		v.other = make(map[string]interface{})
	}
	if err := json.Unmarshal(b, &v.other); err != nil {
		return err
	}
	return nil
}

func (v *Resource) MarshalJSON() ([]byte, error) {
	var err error
	if v.other["Condition"], err = rawMarshal(v.fields.Condition); err != nil {
		return nil, err
	}
	if v.other["AllergyIntolerance"], err = rawMarshal(v.fields.Allergy); err != nil {
		return nil, err
	}
	if v.other["CarePlan"], err = rawMarshal(v.fields.CarePlan); err != nil {
		return nil, err
	}
	if v.other["DocumentReference"], err = rawMarshal(v.fields.Document); err != nil {
		return nil, err
	}
	if v.other["Immunization"], err = rawMarshal(v.fields.Immunization); err != nil {
		return nil, err
	}
	if v.other["MedicationOrder"], err = rawMarshal(v.fields.Medication); err != nil {
		return nil, err
	}
	if v.other["Observation"], err = rawMarshal(v.fields.Observation); err != nil {
		return nil, err
	}
	if v.other["Procedure"], err = rawMarshal(v.fields.Procedure); err != nil {
		return nil, err
	}
	if v.other["DiagnosticReport"], err = rawMarshal(v.fields.Report); err != nil {
		return nil, err
	}
	return json.Marshal(v.other)
}

type Patient struct {
	fields struct {
		Patient string `json:"patient,omitempty"`
		Reference string `json:"reference,omitempty"`
	}
	other map[string]interface{}
}

func (v *Patient) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &v.other); err != nil {
		return err
	}
	return nil
}

func (v *Patient) MarshalJSON() ([]byte, error) {
	var err error
	if v.other["patient"], err = rawMarshal(v.fields.Patient); err != nil {
		return nil, err
	}
	return json.Marshal(v.other)
}

type Subject struct {
	fields struct {
		Patient string `json:"patient,omitempty"`
		Reference string `json:"reference,omitempty"`
	}
	other map[string]interface{}
}

func (v *Subject) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &v.other); err != nil {
		return err
	}
	return nil
}

func (v *Subject) MarshalJSON() ([]byte, error) {
	var err error
	if v.other["subject"], err = rawMarshal(v.fields.Patient); err != nil {
		return nil, err
	}
	return json.Marshal(v.other)
}

type OperationOutcome struct {
	fields struct {
		Issue *Issue `json:"issue,omitempty"`
	}
	other map[string]interface{}
}

func (v *OperationOutcome) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &v.other); err != nil {
		return err
	}
	return nil
}

func (v *OperationOutcome) MarshalJSON() ([]byte, error) {
	var err error
	if v.other["issue"], err = rawMarshal(v.fields.Issue); err != nil {
		return nil, err
	}
	return json.Marshal(v.other)
}

type Issue struct {
	fields struct {
		Issue *Issue `json:"issue,omitempty"`
	}
	other map[string]interface{}
}

func (v *Issue) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &v.other); err != nil {
		return err
	}
	return nil
}

func (v *Issue) MarshalJSON() ([]byte, error) {
	var err error
	if v.other["subject"], err = rawMarshal(v.fields.Issue); err != nil {
		return nil, err
	}
	return json.Marshal(v.other)
}

