package commands

import (
	"fmt"
	"strings"
)

func Mounted() string {
	if len(particionesMontadas) == 0 {
		return fmt.Sprintf("No hay particiones montadas")
	}

	ids := make([]string, len(particionesMontadas))
	for i, pm := range particionesMontadas {
		ids[i] = pm.Id
	}

	return fmt.Sprintf("Particiones montadas: %s", strings.Join(ids, ", "))
}