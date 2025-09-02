# Manual Técnico - Sistema de Archivos EXT2 Simulado

+ Autor: José Emanuel Monzón Lémus
+ Carnet: 202300539
+ Curso: Manejo e implementación de archivos B 
+ Repositorio: https://github.com/0520Jose/MIA_2S2025_P1_202300539.git

---

## Índice

1. [Arquitectura del Sistema](#arquitectura-del-sistema)
2. [Estructuras de Datos](#estructuras-de-datos)
3. [Comandos Implementados](#comandos-implementados)
4. [Referencias y Recursos Adicionales](#referencias-y-recursos-adicionales)
5. [Preguntas Frecuentes (FAQ)](#preguntas-frecuentes-faq)

---

## 1. Arquitectura del Sistema

El sistema simula el funcionamiento de un sistema de archivos EXT2 a través de una aplicación web compuesta por dos módulos principales: **Frontend** y **Backend**. A continuación se describe en detalle la estructura, interacción y flujo de datos entre estos componentes.

---

### 1.1. Frontend

- **Tecnologías:**  
  - React (JavaScript)
  - Vite (herramienta de desarrollo y empaquetado)
- **Funcionalidad:**  
  - Proporciona una interfaz gráfica intuitiva para el usuario.
  - Permite ingresar comandos relacionados con la gestión de discos, particiones, usuarios, archivos y reportes.
  - Visualiza resultados, mensajes de error y reportes generados por el backend.
  - Envía solicitudes HTTP (usualmente POST o GET) al backend para ejecutar acciones.
- **Estructura:**  
  - Componentes para el ingreso de comandos.
  - Ingreso de archivos script tipo .mia.
  - Paneles para mostrar resultado.

---

### 1.2. Backend

- **Tecnologías:**  
  - Go (Golang)
  - API REST (servidor HTTP)
- **Funcionalidad:**  
  - Recibe y procesa comandos enviados por el frontend.
  - Interpreta los comandos y ejecuta la lógica correspondiente (creación de discos, manejo de particiones, gestión de usuarios, manipulación de archivos y directorios, generación de reportes, etc.).
  - Modifica el archivo binario `.mia` que representa el disco virtual y sus estructuras internas (MBR, EBR, inodos, bloques, etc.).
  - Devuelve respuestas estructuradas (JSON) con resultados, mensajes de error o archivos generados (por ejemplo, imágenes de reportes).
- **Estructura:**  
  - Módulo de interpretación de comandos.
  - Módulo de manejo de estructuras de datos del sistema de archivos.
  - Módulo de autenticación y permisos.
  - Módulo de generación de reportes.
  - API REST para comunicación con el frontend.

---

### 1.3. Comunicación Frontend-Backend

- **Protocolo:** HTTP/REST
- **Formato de datos:** JSON para solicitudes y respuestas.
- **Endpoints típicos:**
  - `/api/command` para ejecutar comandos.
  - `/api/report` para solicitar reportes.
  - `/api/login` para autenticación de usuarios.
- **Seguridad:**  
  - Validación de comandos y parámetros.
  - Autenticación de usuarios antes de permitir operaciones sensibles.

---

### 1.4. Flujo de Operación Detallado

1. **Ingreso de Comando:**  
   El usuario utiliza la interfaz web para escribir un comando (por ejemplo, `mkdisk`, `fdisk`, `login`, etc.).
2. **Envío de Solicitud:**  
   El frontend convierte el comando en una solicitud HTTP (usualmente POST) y la envía al backend, incluyendo los parámetros necesarios en formato JSON.
3. **Procesamiento en Backend:**  
   El backend recibe la solicitud, valida los parámetros y ejecuta la lógica correspondiente:
   - Si el comando requiere modificar el archivo `.mia`, abre el archivo y actualiza las estructuras internas.
   - Si el comando es de consulta o reporte, accede a las estructuras y genera la información solicitada.
   - Si el comando es de autenticación, verifica credenciales y permisos.
4. **Respuesta:**  
   El backend envía una respuesta al frontend, que puede incluir:
   - Mensajes de éxito o error.
   - Datos solicitados (por ejemplo, contenido de archivos, información de particiones).
   - Archivos generados (por ejemplo, imágenes de reportes).
5. **Visualización:**  
   El frontend interpreta la respuesta y la muestra al usuario en la interfaz gráfica.

---

### 1.5. Diagrama de Arquitectura

A continuación se presenta un diagrama textual que ilustra la relación entre el frontend y el backend, incluyendo los endpoints principales, el flujo de datos y la integración del archivo `.mia` en el backend. El diagrama combina elementos de componentes y secuencia para mayor claridad.

#### Diagrama de Componentes y Flujo de Datos

```
+-------------------+       HTTP/REST API       +-------------------+
|                   |  ------------------------> |                   |
|    Frontend       |                             |    Backend        |
|  (React + Vite)   |  <------------------------ |  (Go Server)      |
|                   |                             |                   |
+-------------------+                             +-------------------+
         |                                               |
         |                                               |
         v                                               v
+-------------------+                             +-------------------+
| Interfaz de       |                             | API Endpoints:    |
| Usuario:          |                             | - /execute        |
| - Ingreso de      |                             |                   |
|   comandos        |                             +-------------------+
| - Visualización   |                                     |
|   de resultados   |                                     v
+-------------------+                             +-------------------+
                                                 | Procesamiento de  |
                                                 | Comandos:         |
                                                 | - Interpretación  |
                                                 | - Modificación de |
                                                 |   .mia file       |
                                                 +-------------------+
                                                         |
                                                         v
                                                 +-------------------+
                                                 | Archivo .mia:     |
                                                 | - MBR/EBR         |
                                                 | - Superbloque     |
                                                 | - Inodos/Bloques  |
                                                 | - Bitmaps         |
                                                 +-------------------+
```

#### Diagrama de Secuencia (Ejemplo de Comunicación)

```
Usuario -> Frontend: Ingresa comando (e.g., "mkdisk -size=10240")
Frontend -> Backend: POST /execute { "comando": "mkdisk -size=10240" }

---

### 1.6. Ejemplo de Comunicación

**Ejemplo de solicitud desde el frontend:**
```json
POST /api/command
{
  "command": "mkdisk",
  "params": {
    "size": 10240,
    "name": "disco1.mia",
    "path": "/home/user/"
  }
}
```

**Ejemplo de respuesta del backend:**
```json
{
  "status": "success",
  "message": "Disco creado exitosamente.",
  "details": {
    "disk": "disco1.mia",
    "size": 10240
  }
}
```

---

### 1.7. Consideraciones de Diseño

- El sistema está diseñado para ser modular y escalable, permitiendo agregar nuevos comandos y funcionalidades fácilmente.
- La separación entre frontend y backend facilita el mantenimiento y la evolución del sistema.
- El uso de una API REST permite la integración con otras interfaces o herramientas externas en el futuro.

---

## 2. Estructuras de Datos

El sistema simula las principales estructuras del sistema de archivos EXT2:

### MBR (Master Boot Record)
- **Función:** Define las particiones del disco.
- **Campos principales:** tamaño, fecha de creación, particiones (primarias y extendidas).
- **Ubicación:** Al inicio del archivo `.mia`.

### EBR (Extended Boot Record)
- **Función:** Gestiona particiones lógicas dentro de una partición extendida.
- **Campos principales:** estado, tipo, inicio, tamaño, siguiente.
- **Ubicación:** Dentro de la partición extendida.

### Superbloque
- **Función:** Contiene información global del sistema de archivos (cantidad de inodos, bloques, etc.).
- **Campos principales:** número de inodos, número de bloques, tamaño de bloque, etc.

### Inodos
- **Función:** Representan archivos y directorios, almacenan metadatos y punteros a bloques.
- **Campos principales:** tipo, permisos, usuario, grupo, tamaño, punteros a bloques.

### Bloques
- **Función:** Almacenan datos de archivos y directorios.
- **Tipos:** Bloques de datos (contenido de archivos), bloques de carpetas (listado de archivos/directorios).

### Bitmaps
- **Función:** Indican qué inodos y bloques están ocupados o libres.

### Estructuras de Usuarios y Grupos
- **Función:** Gestionan la autenticación y permisos de usuarios y grupos.

> _[Agrega aquí capturas de las estructuras en el código, diagramas de organización interna y ejemplos de cómo se almacenan en el archivo binario.]_

---

## 3. Comandos Implementados

A continuación se listan todos los comandos disponibles, su descripción, parámetros y ejemplos de uso.

### MKDISK
- **Descripción:** Crea un nuevo disco virtual `.mia`.
- **Parámetros:**  
  - `-size`: Tamaño del disco en KB  
  - `-name`: Nombre del archivo  
  - `-path`: Ruta de creación
- **Ejemplo:**
  ```
  mkdisk -size=10240 -name=disco1.mia -path=/home/user/
  ```
- **Efecto:** Inicializa el archivo binario con MBR y espacio para particiones.

---

### RMDISK
- **Descripción:** Elimina un disco virtual.
- **Parámetros:**  
  - `-path`: Ruta del disco a eliminar
- **Ejemplo:**
  ```
  rmdisk -path=/home/user/disco1.mia
  ```
- **Efecto:** Elimina el archivo binario del disco.

---

### FDISK
- **Descripción:** Gestiona particiones en el disco.
- **Parámetros:**  
  - `-size`: Tamaño de la partición  
  - `-unit`: Unidad (K/M)  
  - `-type`: Tipo (P/E/L)  
  - `-fit`: Ajuste (BF/FF/WF)  
  - `-name`: Nombre de la partición  
  - `-path`: Ruta del disco
- **Ejemplo:**
  ```
  fdisk -size=2048 -unit=K -type=P -fit=WF -name=part1 -path=/home/user/disco1.mia
  ```
- **Efecto:** Modifica el MBR/EBR para agregar, eliminar o modificar particiones.

---

### MOUNT
- **Descripción:** Monta una partición para su uso.
- **Parámetros:**  
  - `-name`: Nombre de la partición  
  - `-path`: Ruta del disco
- **Ejemplo:**
  ```
  mount -name=part1 -path=/home/user/disco1.mia
  ```
- **Efecto:** Registra la partición como activa en el sistema.

---

### MOUNTED
- **Descripción:** Lista las particiones actualmente montadas.
- **Parámetros:** Sin parámetros.
- **Ejemplo:**
  ```
  mounted
  ```
- **Efecto:** Muestra las particiones activas.

---

### MKFS
- **Descripción:** Formatea una partición con el sistema de archivos EXT2.
- **Parámetros:**  
  - `-type`: Tipo de formateo (full/fast)  
  - `-id`: Identificador de la partición montada
- **Ejemplo:**
  ```
  mkfs -type=full -id=vda1
  ```
- **Efecto:** Inicializa estructuras internas (superbloque, inodos, bloques).

---

### LOGIN
- **Descripción:** Inicia sesión en el sistema de archivos.
- **Parámetros:**  
  - `-user`: Usuario  
  - `-pass`: Contraseña  
  - `-id`: Identificador de la partición montada
- **Ejemplo:**
  ```
  login -user=admin -pass=123 -id=vda1
  ```
- **Efecto:** Autentica al usuario y permite ejecutar comandos con permisos.

---

### MKGRP
- **Descripción:** Crea un nuevo grupo de usuarios.
- **Parámetros:**  
  - `-name`: Nombre del grupo
- **Ejemplo:**
  ```
  mkgrp -name=grupo1
  ```
- **Efecto:** Agrega un grupo al sistema.

---

### RMGRP
- **Descripción:** Elimina un grupo de usuarios.
- **Parámetros:**  
  - `-name`: Nombre del grupo
- **Ejemplo:**
  ```
  rmgrp -name=grupo1
  ```
- **Efecto:** Elimina el grupo del sistema.

---

### MKUSR
- **Descripción:** Crea un nuevo usuario.
- **Parámetros:**  
  - `-user`: Nombre de usuario  
  - `-pass`: Contraseña  
  - `-grp`: Grupo al que pertenece
- **Ejemplo:**
  ```
  mkusr -user=usuario1 -pass=123 -grp=grupo1
  ```
- **Efecto:** Agrega un usuario al sistema.

---

### RMUSR
- **Descripción:** Elimina un usuario.
- **Parámetros:**  
  - `-user`: Nombre de usuario
- **Ejemplo:**
  ```
  rmusr -user=usuario1
  ```
- **Efecto:** Elimina el usuario del sistema.

---

### MKDIR
- **Descripción:** Crea un nuevo directorio.
- **Parámetros:**  
  - `-path`: Ruta del directorio  
  - `-p`: (Opcional) Crea directorios padres si no existen
- **Ejemplo:**
  ```
  mkdir -path=/carpeta1/carpeta2 -p
  ```
- **Efecto:** Agrega un inodo y bloque de carpeta.

---

### MKFILE
- **Descripción:** Crea un nuevo archivo.
- **Parámetros:**  
  - `-path`: Ruta del archivo  
  - `-size`: Tamaño del archivo  
  - `-cont`: Contenido del archivo  
  - `-p`: (Opcional) Crea directorios padres si no existen
- **Ejemplo:**
  ```
  mkfile -path=/carpeta1/archivo.txt -size=100 -cont="Hola mundo" -p
  ```
- **Efecto:** Agrega un inodo y bloque de datos.

---

### CAT
- **Descripción:** Muestra el contenido de uno o varios archivos.
- **Parámetros:**  
  - `-file`: Ruta(s) de archivo(s)
- **Ejemplo:**
  ```
  cat -file=/carpeta1/archivo.txt
  ```
- **Efecto:** Muestra el contenido del archivo.

---

### CHGRP
- **Descripción:** Cambia el grupo de un usuario.
- **Parámetros:**  
  - `-user`: Usuario  
  - `-grp`: Nuevo grupo
- **Ejemplo:**
  ```
  chgrp -user=usuario1 -grp=grupo2
  ```
- **Efecto:** Modifica el grupo del usuario.

---

### PERMISOS
- **Descripción:** Cambia los permisos de un archivo o carpeta.
- **Parámetros:**  
  - `-path`: Ruta del archivo/carpeta  
  - `-ugo`: Permisos en formato numérico  
  - `-r`: (Opcional) Recursivo
- **Ejemplo:**
  ```
  permisos -path=/carpeta1/archivo.txt -ugo=755 -r
  ```
- **Efecto:** Modifica los permisos del archivo/carpeta.

---

### LIMPIARRUTA
- **Descripción:** Limpia una ruta específica del sistema de archivos.
- **Parámetros:**  
  - `-path`: Ruta a limpiar
- **Ejemplo:**
  ```
  limpiarruta -path=/carpeta1
  ```
- **Efecto:** Elimina el contenido de la ruta especificada.

---

### REPORTES
- **Descripción:** Genera reportes visuales del sistema de archivos.
- **Parámetros:**  
  - `-name`: Tipo de reporte (mbr, disk, inode, block, etc.)  
  - `-path`: Ruta de salida del reporte  
  - `-id`: Identificador de la partición montada
- **Ejemplo:**
  ```
  rep -name=mbr -path=/home/user/reporte_mbr.png -id=vda1
  ```
- **Efecto:** Genera un reporte gráfico del estado del sistema de archivos.

---

> _[Agrega capturas de pantalla de la ejecución de comandos y sus resultados. Puedes incluir ejemplos de entrada/salida para cada comando.]_

---

## 4. Referencias y Recursos Adicionales

- [Documentación oficial EXT2](https://www.kernel.org/doc/html/latest/filesystems/ext2.html)
- [Manual de usuario de la aplicación](./README.md)
- [Enlaces a diagramas y capturas]

---

## 5. Preguntas Frecuentes (FAQ)

**¿Qué sucede si elimino el disco virtual?**  
Se elimina toda la información almacenada en el archivo `.mia`.

**¿Cómo puedo recuperar una partición eliminada?**  
Actualmente no se soporta recuperación automática. Se recomienda realizar respaldos periódicos.

**¿Qué pasa si ingreso un comando incorrecto?**  
El sistema mostrará un mensaje de error indicando el problema.

**¿Puedo crear usuarios y grupos personalizados?**  
Sí, mediante los comandos `mkusr` y `mkgrp`.

**¿Cómo visualizo el estado actual del sistema de archivos?**  
Utiliza el comando `rep` para generar reportes gráficos.

---

## Notas Adicionales

- Si agregas nuevas estructuras, comandos o funcionalidades, documenta aquí.
- Puedes incluir una sección de troubleshooting si lo consideras útil.

---

> _[Agrega aquí cualquier información relevante que consideres necesaria para el entendimiento y uso del sistema. Completa los espacios con capturas, diagramas y ejemplos específicos de tu implementación.]_
