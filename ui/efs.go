package ui

import (
	"embed"
)

//go:embed "html" "static"
var Files embed.FS

// Files embebe los directorios html y static dentro del binario en tiempo
// de compilación, de modo que la aplicación no dependa de la presencia del
// árbol ui/ en el filesystem al ejecutarse. El artefacto de deploy pasa a
// ser un único ejecutable autosuficiente.
//
// La directiva //go:embed de arriba debe quedar pegada a la declaración de
// Files (sin líneas en blanco entre ambas); cualquier comentario adicional
// va antes de la directiva o después de la variable, nunca entre las dos.
