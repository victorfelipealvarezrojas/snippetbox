# Setup de base de datos — Snippetbox (MySQL + Docker)

Documentación del proceso de montaje de la base de datos para el proyecto
Snippetbox (libro *Let's Go*, Alex Edwards), adaptado a **MySQL sobre Docker**.

---

## Cadena de conexión (DSN para Go)

```
web:pass@tcp(localhost:3306)/snippetbox?parseTime=true o web:pass@/snippetbox?parseTime=true"
```

- Usuario de la aplicación: `web` (permisos mínimos)
- `parseTime=true` es **obligatorio**: sin él, Go no convierte los `DATETIME`
  de MySQL a `time.Time` y falla el escaneo de las columnas `created` y `expires`.

### Acceso por terminal (usuario de la app)

```bash
docker exec -it mysql-db mysql -u web -p snippetbox
```

Password: `pass`

> Nota: `pass` es el password de ejemplo del libro. Sirve para desarrollo local;
> cambiar si el entorno deja de ser local.

---

## Paso 1 — Definición del contenedor (docker-compose)

```yaml
services:
  mysql:
    image: mysql:8.0
    container_name: mysql-db
    environment:
      MYSQL_DATABASE: snippetbox
      MYSQL_USER: admin
      MYSQL_PASSWORD: pass12345
      MYSQL_ROOT_PASSWORD: root12345
    ports:
      - "3306:3306"
    volumes:
      - db-data:/var/lib/mysql

volumes:
  db-data:
```

Puntos clave del compose:

- `MYSQL_ROOT_PASSWORD` es obligatoria; sin ella el contenedor no arranca.
- El data dir de MySQL es `/var/lib/mysql` (distinto al de Postgres).
- Las variables `MYSQL_*` solo actúan la **primera vez**, sobre un volumen vacío.

---

## Paso 2 — Levantar el contenedor

```bash
docker compose up -d
```

Al iniciar sobre un volumen vacío, MySQL ejecuta `--initialize`, crea la base
`snippetbox` (desde `MYSQL_DATABASE`) y el usuario `admin`.

> Si se cambia de motor de base de datos (p. ej. de Postgres a MySQL) reusando
> el mismo volumen, **hay que borrar el volumen primero** — cada motor escribe
> una estructura incompatible en su data dir:
>
> ```bash
> docker compose down -v
> docker compose up -d
> ```

Verificar el arranque:

```bash
docker compose logs -f mysql
```

Debe pasar de "Initializing database files" a "ready for connections".

---

## Paso 3 — Crear la tabla `snippets`

Conectado como `admin` o `root`, sobre la base `snippetbox`:

```sql
CREATE TABLE snippets (
    id INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT,
    title VARCHAR(100) NOT NULL,
    content TEXT NOT NULL,
    created DATETIME NOT NULL,
    expires DATETIME NOT NULL
);
```

(Sintaxis MySQL nativa del libro — `AUTO_INCREMENT` y `DATETIME` funcionan
directo, sin traducción.)

---

## Paso 4 — Crear el usuario de la aplicación (`web`)

El usuario con que se conecta la app debe tener **permisos mínimos**
(principio de mínimo privilegio): solo lectura/escritura de filas, sin poder
alterar el esquema ni administrar la base.

Como **root** (los usuarios `MYSQL_USER` no tienen privilegio de `CREATE USER`):

```bash
docker exec -it mysql-db mysql -u root -p
```

Password: `root12345`

Ya como root:

```sql
CREATE USER 'web'@'%' IDENTIFIED BY 'pass';
GRANT SELECT, INSERT, UPDATE, DELETE ON snippetbox.* TO 'web'@'%';
FLUSH PRIVILEGES;
```

> **Detalle Docker (importante):** se usa `@'%'`, no `@'localhost'`.
> `@'localhost'` solo acepta conexiones desde dentro del contenedor; la app en Go
> conecta desde fuera (otra IP de la red Docker) y sería rechazada.
> `@'%'` acepta conexiones desde cualquier host.

---

## Paso 5 — Verificar el usuario y sus permisos

Confirmar que existe con el host correcto:

```sql
SELECT user, host FROM mysql.user WHERE user = 'web';
```

Esperado — una sola fila:

```
+------+------+
| user | host |
+------+------+
| web  | %    |
+------+------+
```

Confirmar los permisos:

```sql
SHOW GRANTS FOR 'web'@'%';
```

Esperado:

```
GRANT USAGE ON *.* TO `web`@`%`
GRANT SELECT, INSERT, UPDATE, DELETE ON `snippetbox`.* TO `web`@`%`
```

(`USAGE ON *.*` es el permiso base de todo usuario — solo permite conectar,
sin privilegios sobre datos. El segundo `GRANT` es el que importa.)

---

## Paso 6 — Probar la conexión como `web`

Prueba final: conectar como `web` desde fuera, tal como lo hará la app.

```bash
docker exec -it mysql-db mysql -u web -p snippetbox
```

Password: `pass`

Ya dentro, verificar acceso de lectura a la tabla:

```sql
SELECT id, title, expires FROM snippets;
```

Si responde (con filas o "empty set"), el usuario conecta y los permisos
funcionan. Base lista para conectar desde Go.

---

## Resumen de usuarios

| Usuario | Host  | Uso                        | Permisos                              |
|---------|-------|----------------------------|---------------------------------------|
| `root`  | —     | Administración             | Todos                                 |
| `admin` | `%`   | Exploración por terminal   | Amplios sobre `snippetbox`            |
| `web`   | `%`   | Aplicación (Go)            | SELECT, INSERT, UPDATE, DELETE        |

---

