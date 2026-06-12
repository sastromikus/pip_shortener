# URL Shortener

Сервис сокращения URL на Go с поддержкой HTTP и gRPC API, нескольких вариантов хранения данных, пользовательской авторизации, асинхронного удаления ссылок, аудита, HTTPS и внутренней статистики.

## Учебный контекст

Проект разработан в рамках курса **«Продвинутый Go-разработчик»** от **Яндекс Практикума**.

Программа создавалась поэтапно на основе учебного задания, состоящего из **28 инкрементов**. За время разработки простой HTTP-сервис с хранением данных в памяти был расширен до многослойного приложения с PostgreSQL, миграциями, middleware, профилированием, gRPC, TLS и корректным завершением фоновых процессов.

В проекте отрабатывались следующие навыки:

- разработка HTTP API на Go;
- маршрутизация и middleware;
- конфигурация через флаги, переменные окружения и JSON-файл;
- хранение данных в памяти, файле и PostgreSQL;
- создание и применение миграций;
- пользовательская авторизация и подпись токенов;
- асинхронная обработка задач;
- аудит событий;
- benchmarking и profiling с `pprof`;
- создание собственного статического анализатора;
- генерация Go-кода;
- работа с generics и `sync.Pool`;
- разработка gRPC API и protobuf Opaque API;
- настройка TLS;
- graceful shutdown HTTP-, gRPC-серверов и фоновых worker-процессов.

## Основные возможности

- сокращение одиночных URL через текстовый и JSON API;
- пакетное сокращение URL;
- перенаправление по короткой ссылке;
- предотвращение дублирования исходных URL;
- получение списка ссылок текущего пользователя;
- асинхронное удаление пользовательских ссылок;
- возврат `410 Gone` для удалённых ссылок;
- поддержка gzip для запросов и ответов;
- проверка соединения с PostgreSQL через `/ping`;
- хранение данных в памяти, JSON-файле или PostgreSQL;
- автоматическое применение миграций PostgreSQL;
- аудит создания и перехода по ссылкам;
- внутренняя статистика по количеству URL и пользователей;
- ограничение доступа к статистике доверенной CIDR-подсетью;
- параллельная работа HTTP и gRPC API;
- HTTPS и TLS для HTTP и gRPC;
- штатное завершение по `SIGINT`, `SIGTERM` и `SIGQUIT`.

## Технологии

- Go 1.26;
- `net/http`;
- `chi`;
- `slog`;
- PostgreSQL;
- `pgx`;
- `goose`;
- gRPC;
- Protocol Buffers;
- protobuf Opaque API;
- `pprof`;
- `go/analysis`.

## Структура проекта

```text
api/grpc/              protobuf-контракт и сгенерированный gRPC-код
cmd/linter/            собственный статический анализатор
cmd/reset/             генератор методов Reset
cmd/shortener/         точка входа приложения
internal/audit/        файловый и HTTP-аудит
internal/auth/         общая подпись и проверка токенов
internal/config/       загрузка конфигурации
internal/grpcserver/   реализация gRPC API
internal/handler/      HTTP handlers и middleware
internal/model/        модели приложения и сгенерированный Reset
internal/pool/         generic-обёртка над sync.Pool
internal/repository/   memory, file и PostgreSQL repositories
internal/service/      бизнес-логика сокращения URL
migrations/            SQL-миграции PostgreSQL
profiles/              профили памяти инкремента 17
```

## Требования

- Go 1.26 или новее;
- PostgreSQL — для режима хранения в базе данных;
- `protoc`, `protoc-gen-go` и `protoc-gen-go-grpc` — только при изменении protobuf-контракта;
- `grpcurl` — для ручной проверки gRPC API.

## Сборка

```bash
go build -o shortener ./cmd/shortener
```

С указанием информации о сборке:

```bash
go build \
  -ldflags="-X main.buildVersion=v1.0.0 -X main.buildDate=$(date +%Y-%m-%d) -X main.buildCommit=$(git rev-parse --short HEAD)" \
  -o shortener \
  ./cmd/shortener
```

При запуске приложение выводит:

```text
Build version: <version>
Build date: <date>
Build commit: <commit>
```

Если значение не задано, выводится `N/A`.

## Запуск

Минимальный запуск:

```bash
./shortener
```

Значения по умолчанию:

```text
HTTP address: localhost:8080
Base URL:     http://localhost:8080
gRPC address: localhost:3200
File storage: storage.json
```

Пример запуска с PostgreSQL:

```bash
./shortener \
  -a localhost:8080 \
  -g localhost:3200 \
  -b http://localhost:8080 \
  -d 'postgres://postgres:password@localhost:5432/shortener?sslmode=disable'
```

При наличии `DATABASE_DSN` используется PostgreSQL. Если DSN не задан, но указан путь к файлу, используется файловое хранилище. Если не задано ни то ни другое, используется хранилище в памяти.

## Конфигурация

Приоритет источников конфигурации:

```text
переменные окружения > флаги > JSON-файл > значения по умолчанию
```

| Назначение | Флаг | Переменная окружения | Поле JSON |
|---|---|---|---|
| HTTP-адрес | `-a` | `SERVER_ADDRESS` | `server_address` |
| Базовый URL | `-b` | `BASE_URL` | `base_url` |
| Файловое хранилище | `-f` | `FILE_STORAGE_PATH` | `file_storage_path` |
| PostgreSQL DSN | `-d`, `-database-dsn`, `-database_dsn` | `DATABASE_DSN` | `database_dsn` |
| Файл аудита | `-audit-file` | `AUDIT_FILE` | `audit_file` |
| URL сервера аудита | `-audit-url` | `AUDIT_URL` | `audit_url` |
| HTTPS/TLS | `-s` | `ENABLE_HTTPS` | `enable_https` |
| Доверенная подсеть | `-t` | `TRUSTED_SUBNET` | `trusted_subnet` |
| gRPC-адрес | `-g`, `-grpc-address` | `GRPC_ADDRESS` | `grpc_address` |
| JSON-конфигурация | `-c`, `-config` | `CONFIG` | — |

Пример JSON-конфигурации:

```json
{
  "server_address": "localhost:8080",
  "base_url": "http://localhost:8080",
  "file_storage_path": "storage.json",
  "database_dsn": "postgres://postgres:password@localhost:5432/shortener?sslmode=disable",
  "audit_file": "audit.log",
  "audit_url": "http://localhost:9000/audit",
  "enable_https": false,
  "trusted_subnet": "192.168.0.0/16",
  "grpc_address": "localhost:3200"
}
```

Для подписи HTTP cookie и gRPC metadata используется переменная:

```text
COOKIE_SECRET
```

Если она не задана, применяется значение для локальной разработки `dev-secret`.

## HTTP API

| Метод | Путь | Назначение |
|---|---|---|
| `POST` | `/` | Сокращение URL из `text/plain` |
| `POST` | `/api/shorten` | Сокращение URL в JSON-формате |
| `POST` | `/api/shorten/batch` | Пакетное сокращение URL |
| `GET` | `/{id}` | Перенаправление на исходный URL |
| `GET` | `/api/user/urls` | Получение ссылок пользователя |
| `DELETE` | `/api/user/urls` | Асинхронное удаление ссылок |
| `GET` | `/ping` | Проверка соединения с PostgreSQL |
| `GET` | `/api/internal/stats` | Внутренняя статистика |

### Сокращение URL

```bash
curl -i \
  -X POST http://localhost:8080/ \
  -H 'Content-Type: text/plain' \
  --data 'https://example.com'
```

Пример ответа:

```text
HTTP/1.1 201 Created
Content-Type: text/plain

http://localhost:8080/59N1F2fL
```

### JSON API

```bash
curl -i \
  -X POST http://localhost:8080/api/shorten \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com"}'
```

```json
{
  "result": "http://localhost:8080/59N1F2fL"
}
```

### Пакетное сокращение

```bash
curl -i \
  -X POST http://localhost:8080/api/shorten/batch \
  -H 'Content-Type: application/json' \
  -d '[
    {"correlation_id":"1","original_url":"https://example.com"},
    {"correlation_id":"2","original_url":"https://go.dev"}
  ]'
```

### Переход по короткой ссылке

```bash
curl -i http://localhost:8080/59N1F2fL
```

При успешном запросе возвращается `307 Temporary Redirect`. Для удалённой ссылки возвращается `410 Gone`.

## Авторизация

HTTP API использует подписанную cookie `user_id`. Если корректная cookie отсутствует, сервер автоматически создаёт новый идентификатор пользователя и возвращает cookie клиенту.

gRPC передаёт тот же подписанный токен через metadata:

```text
authorization: <signed-token>
```

Методы `ShortenURL` и `ListUserURLs` требуют авторизацию. `ExpandURL`, как и HTTP `GET /{id}`, доступен без авторизации.

## Внутренняя статистика

```http
GET /api/internal/stats
X-Real-IP: 192.168.1.10
```

Ответ:

```json
{
  "urls": 100,
  "users": 25
}
```

IP из заголовка `X-Real-IP` должен входить в подсеть `TRUSTED_SUBNET`. При пустой, некорректной или неподходящей подсети сервер возвращает `403 Forbidden`.

## gRPC API

Сервис:

```text
shortener.v1.ShortenerService
```

Методы:

```text
ShortenURL
ExpandURL
ListUserURLs
```

Проверка списка сервисов:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto api/grpc/shortener.proto \
  localhost:3200 list
```

Описание сервиса:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto api/grpc/shortener.proto \
  localhost:3200 describe shortener.v1.ShortenerService
```

Анонимное раскрытие ссылки:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto api/grpc/shortener.proto \
  -d '{"id":"59N1F2fL"}' \
  localhost:3200 shortener.v1.ShortenerService/ExpandURL
```

Сокращение URL с авторизацией:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto api/grpc/shortener.proto \
  -H 'authorization: <signed-token>' \
  -d '{"url":"https://example.com"}' \
  localhost:3200 shortener.v1.ShortenerService/ShortenURL
```

## HTTPS и TLS

HTTPS включается флагом:

```bash
./shortener -s
```

или переменной окружения:

```text
ENABLE_HTTPS=true
```

При включении TLS приложение создаёт самоподписанный сертификат и использует согласованную TLS-конфигурацию для HTTP и gRPC.

Самоподписанный сертификат предназначен для учебного и локального запуска. Для production-среды необходимо использовать сертификат от доверенного центра сертификации.

## Аудит

Аудит поддерживает два типа получателей:

- файл, заданный через `AUDIT_FILE` или `-audit-file`;
- HTTP endpoint, заданный через `AUDIT_URL` или `-audit-url`.

Аудитом покрыты:

- создание короткой ссылки;
- переход по короткой ссылке.

Пример события:

```json
{
  "ts": 1710000000,
  "action": "shorten",
  "user_id": "user-id",
  "url": "https://example.com"
}
```

События передаются через ограниченную асинхронную очередь, чтобы медленный получатель не задерживал HTTP-ответ.

## PostgreSQL и миграции

Для работы с PostgreSQL используется `pgx`, для миграций — `goose`.

При запуске с непустым `DATABASE_DSN` приложение автоматически:

1. подключается к PostgreSQL;
2. проверяет состояние схемы;
3. применяет миграции из каталога `migrations/`;
4. запускает HTTP и gRPC серверы.

Миграции создают таблицы URL и связей пользователя со ссылками, уникальное ограничение исходного URL и признак удаления.

## Graceful shutdown

Приложение штатно обрабатывает:

```text
SIGINT
SIGTERM
SIGQUIT
```

При завершении:

- прекращается приём новых HTTP-запросов;
- корректно останавливается gRPC-сервер;
- завершается очередь асинхронного удаления;
- закрываются audit observers;
- закрывается соединение с PostgreSQL;
- ожидается завершение фоновых goroutine.

## Тестирование

```bash
go test ./...
```

С detector состояния гонки:

```bash
go test -race ./...
```

Статический анализ:

```bash
go vet ./...
```

## Profiling — инкремент 17

### Benchmark

Для профилирования использовался handler-level benchmark `BenchmarkPOSTShortenTextPlain`.

```sh
go test ./internal/handler -run ^$ -bench BenchmarkPOSTShortenTextPlain -benchmem -count=5
```

### Base profile

Базовый профиль снимался на версии до оптимизации logging middleware.

```sh
go test ./internal/handler -run ^$ -bench BenchmarkPOSTShortenTextPlain -benchmem -count=5 -memprofile profiles/base.pprof
```

### Result profile

Итоговый профиль снимался на версии после оптимизации logging middleware и перехода со старого пути логирования на `slog`.

```sh
go test ./internal/handler -run ^$ -bench BenchmarkPOSTShortenTextPlain -benchmem -count=5 -memprofile profiles/result.pprof
```

### Diff

```sh
go tool pprof -top -nodecount=25 -diff_base=profiles/base.pprof profiles/result.pprof
```

Output:

```text
File: handler.test.exe
Type: alloc_space
Showing nodes accounting for 41.14MB, 2.78% of 1482MB total

      flat  flat%   sum%        cum   cum%
 -125.53MB  8.47%  8.47%  -125.53MB  8.47%  github.com/sirupsen/logrus.(*Entry).WithFields
  -99.02MB  6.68% 15.15%   -99.02MB  6.68%  github.com/sirupsen/logrus.(*Entry).Dup
  -65.01MB  4.39% 19.54%   -81.51MB  5.50%  github.com/sirupsen/logrus.(*TextFormatter).Format
   54.51MB  3.68% 15.86%    54.51MB  3.68%  net/url.parse
   44.01MB  2.97% 12.89%    44.01MB  2.97%  net/http.(*Request).WithContext
   40.51MB  2.73% 10.16%    40.51MB  2.73%  net/textproto.MIMEHeader.Set
      35MB  2.36%  7.79%       48MB  3.24%  net/url.(*URL).joinPath
   32.51MB  2.19%  5.60%    32.51MB  2.19%  net/http.Header.Clone
  -19.22MB  1.30%  6.90%   -19.22MB  1.30%  github.com/sastromikus/pip_shortener/internal/repository.(*MemoryRepository).Put
   16.50MB  1.11%  5.78%    35.01MB  2.36%  github.com/sastromikus/pip_shortener/internal/handler.BenchmarkGETFollow
   16.50MB  1.11%  4.67%    16.50MB  1.11%  net/http.MaxBytesReader
      14MB  0.94%  3.73%       18MB  1.21%  net/http/httptest.(*ResponseRecorder).Result
     -14MB  0.94%  4.67%      -14MB  0.94%  github.com/sirupsen/logrus.(*Logger).releaseEntry
      12MB  0.81%  3.86%       21MB  1.42%  crypto/internal/fips140/hmac.New
      11MB  0.74%  3.12%       11MB  0.74%  net/http.readCookies
      11MB  0.74%  2.38%       53MB  3.58%  github.com/sastromikus/pip_shortener/internal/handler/middleware.verifyCookie
  -10.26MB  0.69%  3.07%   -10.26MB  0.69%  github.com/sastromikus/pip_shortener/internal/repository.(*MemoryRepository).AddUserURL
      10MB  0.68%  2.39%    10.52MB  0.71%  io.ReadAll
    9.50MB  0.64%  1.75%     9.50MB  0.64%  bytes.NewReader
       9MB  0.61%  1.15%        9MB  0.61%  net/http.(*Request).SetPathValue
       9MB  0.61% 0.069%        9MB  0.61%  context.WithValue
       8MB  0.54%  0.61%        8MB  0.54%  encoding/hex.EncodeToString
    7.50MB  0.51%  1.12%       40MB  2.70%  github.com/sastromikus/pip_shortener/internal/handler/middleware.signCookie
    6.50MB  0.44%  2.06%    40.62MB  2.74%  github.com/sastromikus/pip_shortener/internal/handler.BenchmarkPOSTShortenTextPlain
       6MB   0.4%  2.90%    83.04MB  5.60%  github.com/sastromikus/pip_shortener/internal/handler.handleShorten
```

Основное снижение аллокаций получено в logging middleware. В базовом профиле значительная часть памяти расходовалась на старый путь логирования через `logrus`: создание `Entry`, вызовы `WithFields`, `Dup` и форматирование текстового вывода.

После оптимизации эти аллокации исчезли из горячего пути или стали заметно меньше:

```text
-125.53MB  github.com/sirupsen/logrus.(*Entry).WithFields
 -99.02MB  github.com/sirupsen/logrus.(*Entry).Dup
 -65.01MB  github.com/sirupsen/logrus.(*TextFormatter).Format
```

Положительные значения в diff остались в основном в стандартной HTTP-инфраструктуре и коде обработки запроса:

- `net/url.parse`;
- `net/http.(*Request).WithContext`;
- `net/textproto.MIMEHeader.Set`;
- `net/http.Header.Clone`;
- `net/http/httptest.(*ResponseRecorder).Result`;
- cookie-auth middleware.

Эти аллокации ожидаемы для handler benchmark: каждый прогон создаёт HTTP-запрос, recorder, headers, cookie и проходит через router/middleware stack.

Итог: оптимизация уменьшила расходы на логирование, а основная оставшаяся стоимость относится к HTTP stack, cookie-auth и построению URL.

## Дополнительные учебные компоненты

Помимо основного сервиса, в проект входят:

- собственный статический анализатор, запрещающий `panic`, `log.Fatal` и `os.Exit` вне допустимых мест;
- генератор методов `Reset()` на основе `go/ast`;
- generic pool для объектов с методом `Reset()`;
- примеры использования HTTP handlers в `example_test.go`;
- build metadata, передаваемые через `ldflags`.

## Ограничения

- встроенный TLS использует самоподписанный сертификат;
- значение `dev-secret` нельзя использовать в production;
- внутренний endpoint статистики рассчитан на работу за доверенным reverse proxy, который устанавливает `X-Real-IP`;
- gRPC reflection не включён, поэтому `grpcurl` следует запускать с указанием `.proto` файла.
