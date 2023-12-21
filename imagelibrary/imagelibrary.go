package imagelibrary

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/solanashuffle/backend/utility"
)

var (
	library = []string{}
)

func init() {
	files, err := ioutil.ReadDir("./imagelibrary")
	if err != nil {
		return
		panic(err)
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			var arr []string
			j, err := ioutil.ReadFile(fmt.Sprintf("./imagelibrary/%s", f.Name()))
			if err != nil {
				panic(err)
			}
			json.Unmarshal(j, &arr)
			library = append(library, arr...)
		}
	}
}

func Random() string {
	return library[utility.RandomInt(0, len(library)-1)]
}
