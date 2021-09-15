package fhir_categories

import (
	"encoding/json"
	"log"
)

func (v *Observation) GetFilteredCategory(bytes []byte) []byte {
	var root Root
	log.Println("initializing")
	if err := json.Unmarshal(bytes, &root); err != nil {
		log.Println(err)
	}

	jsonObject, _ := json.MarshalIndent(&root, "", "    ")
	return jsonObject
}

type Observation struct {
	fields struct {
		Subject *Subject `json:"subject,omitempty"`
	}
	other map[string]interface{}
}


func (v *Observation) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &v.fields.Subject); err != nil {
		return err
	}
	if v.other == nil {
		v.other = make(map[string]interface{})
	}
	if err := json.Unmarshal(b, &v.other); err != nil {
		return err
	}
	log.Println("Setting the subject observation data to null")

	v.fields.Subject = nil

	return nil
}

func (v *Observation) MarshalJSON() ([]byte, error) {
	var err error
	if v.other["subject"], err = rawMarshal(v.fields.Subject); err != nil {
		return nil, err
	}
	return json.Marshal(v.other)
}