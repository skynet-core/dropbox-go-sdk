package tests_test

import (
	"encoding/json"
	"log"
	"testing"

	"github.com/skynet-core/dropbox-go-sdk/dropbox"
	"github.com/skynet-core/dropbox-go-sdk/dropbox/file_properties"
	"github.com/stretchr/testify/assert"
)

func TestMarshal(t *testing.T) {
	data, err := json.Marshal(&file_properties.TemplateFilterBase{
		Tagged:     dropbox.Tagged{Tag: file_properties.TemplateFilterFilterSome},
		FilterSome: []string{"hello"},
	})
	assert.NoError(t, err)
	log.Println(string(data))
	fieldTemplates := []*file_properties.PropertyFieldTemplate{
		{
			Name: "test",
			Type: &file_properties.PropertyType{
				Tagged: dropbox.Tagged{Tag: "string"},
			},
		},
	}
	data, err = json.Marshal(fieldTemplates)
	assert.NoError(t, err)
	log.Println(string(data))
}
