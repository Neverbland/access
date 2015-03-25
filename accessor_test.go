package access

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPropertyPath(t *testing.T) {
	assert := assert.New(t)

	assert.NotPanics(func() { New("field[0][1].key[2]") })

	assert.Panics(func() { New("f.") })
	assert.Panics(func() { New(".f") })
	assert.Panics(func() { New("f[ 1]") })
	assert.Panics(func() { New([]string{"field", "0", "key"}) })
	assert.Panics(func() { New("") })
	assert.Panics(func() { New("[1]field") })     //dot required before field access
	assert.Panics(func() { New("[hello]field") }) //not numeric index

	assert.Equal("field[0][1].key[2]", New("field[0][1].key[2]").String())
}

type glossary struct {
	GlossDiv *glossDiv `json:"GlossDiv"`
	Title    string    `json:"title"`
}

type MSI map[string]interface{}

type glossEntry MSI

func (g glossEntry) Field(field string) (interface{}, error) {
	return Read(field, MSI(g))
}

func (g *glossEntry) SetField(field string, val interface{}) error {
	return Write(field, (*MSI)(g), val)
}

type glossDiv struct {
	Title     string        `json:"title"`
	GlossList []*glossEntry `json:"GlossList"`
}

func (g glossDiv) Index(index int) (interface{}, error) {
	return Read(index, &g.GlossList)
}

func (g *glossDiv) SetIndex(index int, val interface{}) error {
	return Write(index, &g.GlossList, val)
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
		},
		"glossary2": {}
	}`)

func TestUsage(t *testing.T) {
	assert := assert.New(t)

	data := &(map[string]glossary{})

	assert.NoError(json.Unmarshal(jsonStr, data))

	assert.Equal("ISO 8879:1986", MustRead("glossary.gloss_div.GlossList[0].Abbrev", data))
	assert.NoError(Write("glossary.gloss_div.GlossList[0].Abbrev", data, "ISO 0000:0000"))
	assert.Equal("ISO 0000:0000", MustRead("glossary.gloss_div.GlossList[0].Abbrev", data))

	//using IndexReader /IndexWriter
	assert.Equal("ISO 8879:1986", MustRead("glossary.gloss_div[1].Abbrev", data))

	assert.NoError(Write("glossary.gloss_div[1].Abbrev", data, "ISO 0000:0000"))

	//empty elements
	assert.Nil(MustRead("glossary.gloss_div[2].Abbrev", data))
	assert.Nil(MustRead("glossary.gloss_div.GlossList[2].Abbrev", data))

	assert.NoError(Write("glossary3", data, glossary{Title: "Test"}))
	assert.Equal("Test", MustRead("glossary3.title", data))

	assert.NoError(Write("glossary3.title", data, ""))
	assert.Equal("", MustRead("glossary3.title", data))

	entry := &glossEntry{"Abbrev": "ISO 1111:1111", "V": "Test"}

	assert.NoError(Write("glossary.gloss_div[2]", data, entry))

	assert.Equal(entry, MustRead("glossary.gloss_div[2]", data))

	assert.NoError(Write("glossary.gloss_div.GlossList[3]", data, entry))

	assert.Equal(entry, MustRead("glossary.gloss_div.GlossList[3]", data))
}

func TestMSI(t *testing.T) {
	assert := assert.New(t)

	data := map[string]interface{}{
		"hello": map[string]interface{}{
			"world": "!",
		},
	}

	assert.Equal("!", MustRead("hello.world", &data))
	assert.NoError(Write("hello.world", &data, glossary{Title: "!"}))

	assert.Equal("!", MustRead("hello.world.title", data))

	assert.NoError(Write("hello.world.title", &data, ""))
	assert.Equal("", MustRead("hello.world.title", data))

}
