# Subtree paths en el `ServeMux` de Go

## La regla en una línea

Un patrón de ruta que **termina en `/`** (como `"/"` o `"/static/"`) es un
*subtree path*. Hace match con cualquier request cuyo path **empiece** con ese
prefijo, sin importar lo que venga después.

Mentalmente, se puede leer como si tuviera un wildcard al final:

```
"/static/"   ≈   "/static/**"
```

## Cómo se comporta cada patrón

| Patrón registrado | Hace match con | No hace match con |
|---|---|---|
| `"/static/"` | `/static/`, `/static/css/main.css`, `/static/a/b/c` | `/static` (sin slash → ver redirect) |
| `"/"` | **todo** (todo path empieza con `/`) | nada; es el catch-all |
| `"/snippet/view"` | solo `/snippet/view` (exacto) | `/snippet/view/123` |

## Los dos matices donde se rompe la intuición

### 1. Sin slash final = match exacto (fixed path)

`"/snippet/view"` dispara **únicamente** con esa ruta exacta.
`/snippet/view/123` no cae ahí.

- Termina en `/` → subtree (match por prefijo).
- No termina en `/` → fixed path (match exacto).

### 2. Gana el patrón más específico, no el primero registrado

Con `"/"` y `"/static/"` registrados en el mismo mux, un request a
`/static/css/main.css` lo atiende `"/static/"`, no `"/"`.

El patrón más largo / específico tiene prioridad. `"/"` solo captura lo que
ningún otro patrón reclamó — de ahí que sirva como home / catch-all.

## Detalle del redirect

Si está registrado `"/static/"` y alguien pide `/static` (sin slash), el mux
responde un **`301`** redirigiendo a `/static/`. La versión con slash es la
canónica.

## Restringir un subtree path con `{$}`

Para impedir que un subtree path se comporte como si tuviera un wildcard al
final, se agrega la secuencia especial `{$}` al final del patrón —
`"/{$}"` o `"/static/{$}"`.

`"/{$}"` significa literalmente: *match de una sola barra, seguida de nada
más*. Solo dispara cuando el path del URL es **exactamente** `/`.

Esto sirve para que el home handler deje de actuar como catch-all:

```go
// Antes: "/" captura TODO lo que ningún otro patrón reclame.
mux.HandleFunc("/", home)

// Después: "/{$}" captura ÚNICAMENTE el path exacto "/".
mux.HandleFunc("/{$}", home)
```

Con `"/{$}"`, un request a `/algo-inexistente` ya no cae en `home`: el mux
responde `404` en vez de servir la home. Cada ruta queda acotada a lo suyo.

La misma lógica aplica a cualquier prefijo: `"/static/{$}"` matchea solo
`/static/` exacto, sin los archivos que cuelgan debajo.

> `{$}` es parte del enrutador de Go 1.22+.

## Nota de versión (Go 1.22+)

El enrutador nuevo mantiene todo el comportamiento de trailing slash descrito
arriba, y además agrega:

- **Wildcards nombrados:** `"/static/{path...}"` hace lo mismo que el subtree
  path, pero de forma más explícita y legible y ademas me permite acceder a toda la parte comodín a través del
Método r.PathValue() en sus controladores. En este ejemplo, podría obtener el valor comodín.
para {ruta...} llamando a r.PathValue("ruta").
- **Método en el patrón:** `"GET /static/"` restringe por verbo HTTP.

En el recorrido de *Let's Go Further* (Alex Edwards) los subtree paths aplican
igual; los wildcards con nombre aparecen más adelante.


## El `DefaultServeMux`: qué es y por qué evitarlo

Cuando se usan las funciones a nivel de paquete —`http.HandleFunc(...)`,
`http.Handle(...)`— no se registra en un mux propio, sino en el
**`DefaultServeMux`**: una instancia de `ServeMux` que `net/http` crea y guarda
como variable global del paquete.

```go
func main() {
    // Registra en el DefaultServeMux (variable global del paquete).
    http.HandleFunc("/", home)
    http.HandleFunc("/snippet/view", snippetView)
    http.HandleFunc("/snippet/create", snippetCreate)

    // El nil de acá significa: "no paso handler, usa el DefaultServeMux".
    http.ListenAndServe(":4000", nil)
}
```

`http.HandleFunc` por dentro hace `DefaultServeMux.HandleFunc(...)`, y
`ListenAndServe` con `nil` como segundo argumento cae al `DefaultServeMux`. El
código de arriba es equivalente exacto a registrar en un mux propio:

```go
func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", home)
    mux.HandleFunc("/snippet/view", snippetView)
    mux.HandleFunc("/snippet/create", snippetCreate)

    http.ListenAndServe(":4000", mux)   // mux explícito, no nil
}
```

### Por qué conviene el mux propio

No es cosmético. El `DefaultServeMux` es una **global**, y cualquier paquete que
importe el programa puede registrar rutas en él sin que se vea. El caso clásico:
si una dependencia (directa o transitiva) importa `net/http/pprof` o `expvar`,
esos paquetes registran handlers en el `DefaultServeMux` desde su `init()`. El
resultado son endpoints expuestos que nunca se declararon.

Con un `ServeMux` propio vía `http.NewServeMux()`, el registro de rutas queda
local a esa variable: nadie externo puede inyectar handlers ahí. Control
explícito y superficie de ataque acotada.

> Regla práctica: usar siempre un mux propio pasado explícitamente a
> `ListenAndServe`, no el `nil`.


## Routers de terceros

El routing por wildcards y por método que venimos usando es relativamente
nuevo en Go — recién entró a la librería estándar en Go **1.22**. Es una muy
buena adición, pero hay casos donde el `ServeMux` estándar no alcanza. A la
fecha (verificado a agosto de 2026, siguen vigentes) no soporta:

- Enviar respuestas **404 Not Found** y **405 Method Not Allowed**
  personalizadas. Hay propuestas abiertas al respecto desde 2024, e incluso una
  nueva de mayo de 2026 pidiendo poder sobrescribir los handlers internos del
  `ServeMux` — pero ninguna se ha integrado todavía.
- Usar **expresiones regulares** en los patrones o wildcards.
- Matchear **múltiples métodos HTTP** en una sola declaración de ruta.
- Soporte automático de requests **OPTIONS**.
- Enrutar según cosas inusuales, como **headers** del request HTTP.

### El detalle del 404 vs 405

El punto del 404/405 es el más filoso. Con el `ServeMux` de 1.22+ **no** se
puede usar un catch-all `"/"` para devolver un 404 propio, porque hacerlo
inhibe el envío automático del 405. Es uno o el otro. Si se necesitan ambas
respuestas personalizadas y que cumplan bien con las specs de HTTP —caso
típico: una API JSON de uso público— conviene ir directo a un router de
terceros.

### Cuáles usar

Los routers recomendados por Edwards son **httprouter**, **chi**, **flow** y
**gorilla/mux**. La guía práctica de su blog:

- **404/405 custom + apego estricto a specs HTTP** (ej. API JSON pública):
  httprouter o flow.
- **Mucho middleware específico por ruta:** chi o flow, por su agrupación de
  middleware.
- **En otro caso:** empezar con `http.ServeMux` y migrar a un router de
  terceros solo si aparece una feature o comportamiento puntual que se necesite.

> Comparación completa y criterio de elección:
> https://www.alexedwards.net/blog/which-go-router-should-i-use