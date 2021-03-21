package file_properties

import "encoding/json"

// MarshalJSON ...
func (pt PropertyType) MarshalJSON() ([]byte, error) {
	return json.Marshal(pt.Tagged.Tag)
}
