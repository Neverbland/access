package access

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPropertyPathConstruct(t *testing.T) {
	assert := assert.New(t)

	assert.NotPanics(func() { New("field[0][1].key[2]") })

	assert.Panics(func() { New("f.") })
	assert.Panics(func() { New(".f") })
	assert.Panics(func() { New("f[ 1]") })
	assert.Panics(func() { New("[1]field") }) //dot required before field access
}

type TestStruct struct {
	Glossary struct {
		GlossDiv glossDiv `json:"GlossDiv"`
		Title    string   `json:"title"`
	} `json:"glossary"`
}

type glossEntry struct {
	Abbrev   string `json:"Abbrev"`
	Acronym  string `json:"Acronym"`
	GlossDef struct {
		GlossSeeAlso []string `json:"GlossSeeAlso"`
		Para         string   `json:"para"`
	} `json:"GlossDef"`
	GlossSee  string `json:"GlossSee"`
	GlossTerm string `json:"GlossTerm"`
	ID        string `json:"ID"`
	SortAs    string `json:"SortAs"`
}

type glossDiv struct {
	Title     string       `json:"title"`
	GlossList []glossEntry `json:"GlossList"`
}

func (g *glossDiv) ReadIndexPath(path PropertyPath) (interface{}, error) {
	return path.Read(&g.GlossList)
}

func (g *glossDiv) WriteIndexPath(path PropertyPath, val interface{}) error {
	return path.Write(&g.GlossList, val)
}

var jsonStr = []byte(
	`{
		"glossary": {
			"title": "example glossary",
			"GlossDiv": {
				"title": "S",
				"GlossList": [
					{
						"ID": "SGML",
						"SortAs": "SGML",
						"GlossTerm": "Standard Generalized Markup Language",
						"Acronym": "SGML",
						"Abbrev": "ISO 8879:1986",
						"GlossDef": {
							"para": "A meta-markup language, used to create markup languages such as DocBook.",
							"GlossSeeAlso": ["GML", "XML"]
						},
						"GlossSee": "markup"
					},
					{
						"ID": "SGML",
						"SortAs": "SGML",
						"GlossTerm": "Standard Generalized Markup Language",
						"Acronym": "SGML",
						"Abbrev": "ISO 8879:1986",
						"GlossDef": {
							"para": "A meta-markup language, used to create markup languages such as DocBook.",
							"GlossSeeAlso": ["GML", "XML"]
						},
						"GlossSee": "markup"
					}
				]
			}
		}
	}`)

func TestUsage(t *testing.T) {
	assert := assert.New(t)

	data := &TestStruct{}
	err := json.Unmarshal(jsonStr, data)
	assert.Nil(err)

	assert.Equal("ISO 8879:1986", MustRead("glossary.gloss_div.GlossList[0].abbrev", data))
	err = Write("glossary.gloss_div.GlossList[0].abbrev", data, "ISO 0000:0000")
	assert.Nil(err)

	assert.Equal("ISO 0000:0000", MustRead("glossary.gloss_div.GlossList[0].abbrev", data))

	//using IndexReader /IndexWriter
	assert.Equal("ISO 8879:1986", MustRead("glossary.gloss_div[1].abbrev", data))

	err = Write("glossary.gloss_div[1].abbrev", data, "ISO 0000:0000")
	assert.Nil(err)

	//empty elements
	assert.Equal(nil, MustRead("glossary.gloss_div[2].abbrev", data))
	assert.Equal(nil, MustRead("glossary.gloss_div.GlossList[2].abbrev", data))

	//allocation using IndexWriter

	entry := glossEntry{Abbrev: "ISO 1111:1111"}

	err = Write("glossary.gloss_div[2]", data, entry)
	assert.Nil(err)

	assert.Equal(entry, MustRead("glossary.gloss_div[2]", data))
	assert.Equal("ISO 1111:1111", MustRead("glossary.gloss_div[2].abbrev", data))

	//allocation using slice
	err = Write("glossary.gloss_div.GlossList[3]", data, entry)
	assert.Nil(err)

	assert.Equal(entry, MustRead("glossary.gloss_div[3]", data))
	assert.Equal("ISO 1111:1111", MustRead("glossary.gloss_div.GlossList[3].abbrev", data))

}
