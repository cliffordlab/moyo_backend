package fhir_categories

import (
	"encoding/json"
	"log"
)

func (v *Medication) GetFilteredCategory(bytes []byte) []byte {
	var root Root
	log.Println("initializing")
	if err := json.Unmarshal(bytes, &root); err != nil {
		log.Println(err)
	}

	jsonObject, _ := json.MarshalIndent(&root, "", "    ")
	return jsonObject
}

type Medication struct {
	fields struct {
		Patient *Patient `json:"patient,omitempty"`
	}
	other map[string]interface{}
}


func (v *Medication) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &v.fields.Patient); err != nil {
		return err
	}
	if v.other == nil {
		v.other = make(map[string]interface{})
	}
	if err := json.Unmarshal(b, &v.other); err != nil {
		return err
	}
	log.Println("Setting the patient medication data to null")

	v.fields.Patient = nil

	return nil
}

func (v *Medication) MarshalJSON() ([]byte, error) {
	var err error
	if v.other["patient"], err = rawMarshal(v.fields.Patient); err != nil {
		return nil, err
	}
	return json.Marshal(v.other)
}