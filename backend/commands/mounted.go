package commands

import (
	"fmt"
	"strings"
)

func Mounted() {
	if len(particionesMontadas) == 0 {
		fmt.Println("No hay particiones montadas")
		return
	}

	ids := make([]string, len(particionesMontadas))
	for i, pm := range particionesMontadas {
		ids[i] = pm.Id
	}

	fmt.Println(strings.Join(ids, ", "))
}