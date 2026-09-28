package structs

import (
	"fmt"
	"strings"
)

type Grupo struct {
	GID    int
	Nombre string
}

type Usuario struct {
	UID        int
	Grupo      string
	Nombre     string
	Contrasena string
}

type UsersTXT struct {
	Grupos   []Grupo
	Usuarios []Usuario
}

func (u *UsersTXT) ToString() string {
	var sb strings.Builder
	for _, g := range u.Grupos {
		sb.WriteString(fmt.Sprintf("%d,G,%s\n", g.GID, g.Nombre))
	}
	for _, usr := range u.Usuarios {
		sb.WriteString(fmt.Sprintf("%d,U,%s,%s,%s\n", usr.UID, usr.Grupo, usr.Nombre, usr.Contrasena))
	}
	return sb.String()
}

