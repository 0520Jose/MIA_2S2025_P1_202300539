package structs

import (
	"fmt"
	"strings"
    "strconv"
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

func NuevoUsersTXT() UsersTXT {
	return UsersTXT{
		Grupos:   []Grupo{{GID: 1, Nombre: "root"}},
		Usuarios: []Usuario{{UID: 1, Grupo: "root", Nombre: "root", Contrasena: "123"}},
	}
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

func (u *UsersTXT) FromString(data string) {
    u.Grupos = []Grupo{}
    u.Usuarios = []Usuario{}
    
    lines := strings.Split(data, "\n")
    for _, line := range lines {
        if line == "" {
            continue
        }
        parts := strings.Split(line, ",")
        if len(parts) < 2 {
            continue
        }
        if parts[1] == "G" && len(parts) == 3 {
            gid, _ := strconv.Atoi(parts[0])
            nombre := parts[2]
            u.Grupos = append(u.Grupos, Grupo{GID: gid, Nombre: nombre})
        } else if parts[1] == "U" && len(parts) == 5 {
            uid, _ := strconv.Atoi(parts[0])
            grupo := parts[2]
            nombre := parts[3]
            contrasena := parts[4]
            u.Usuarios = append(u.Usuarios, Usuario{UID: uid, Grupo: grupo, Nombre: nombre, Contrasena: contrasena})
        }
    }
}


func (u *UsersTXT) AddUsuario(nombre, grupo, contrasena string) {
	uid := len(u.Usuarios) + 1
	u.Usuarios = append(u.Usuarios, Usuario{UID: uid, Grupo: grupo, Nombre: nombre, Contrasena: contrasena})
}

func (u *UsersTXT) AddGrupo(nombre string) {
	gid := len(u.Grupos) + 1
	u.Grupos = append(u.Grupos, Grupo{GID: gid, Nombre: nombre})
}

func (u *UsersTXT) GetUsuario(nombre string) *Usuario {
	for _, usr := range u.Usuarios {
		if usr.Nombre == nombre {
			return &usr
		}
	}
	return nil
}
