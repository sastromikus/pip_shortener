# go-musthave-shortener-tpl

Шаблон репозитория для трека «Сервис сокращения URL».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-shortener-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Profiling (iter17)

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
