package models

type Addresses struct {
    Components   []map[string]string
    Country_code string `json:"country_code"`
	Formatted string `json:"Formatted"`
	Point struct {}
	URI string `json:"uri"`
}



//Пример {"Components": [{"kind": null, "name": "Pamyatnik"}, {"kind": null, "name": "Monuments Karelian granite"}], "country_code": null, "formatted": null, "point": {"lat": 59.941635, "lon": 30.427217}, "uri": "ymapsbm1://org?oid=1103120093"}