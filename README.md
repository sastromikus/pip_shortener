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

```sh
go test ./internal/handler -run ^$ -bench BenchmarkPOST_Shorten_TextPlain -benchmem -count=5
```

### Base profile

```sh
go test ./internal/handler -run ^$ -bench BenchmarkPOST_Shorten_TextPlain -benchmem -count=5 -memprofile profiles/base.pprof
```

### Result profile

```sh
go test ./internal/handler -run ^$ -bench BenchmarkPOST_Shorten_TextPlain -benchmem -count=5 -memprofile profiles/result.pprof
```

### Diff

```sh
go tool pprof -top -nodecount=25 -diff_base=profiles/base.pprof profiles/result.pprof
```

Output:

```text
File: handler.test.exe
Build ID: C:\Users\PIPOBU~1\AppData\Local\Temp\go-build3235056443\b001\handler.test.exe2026-05-04 23:55:55.4892091 +0300 MSK
Type: alloc_space
Time: 2026-05-04 23:56:04 MSK
Showing nodes accounting for 1.30GB, 92.01% of 1.41GB total
Dropped 47 nodes (cum <= 0.01GB)
Showing top 25 nodes out of 81
      flat  flat%   sum%        cum   cum%
    0.53GB 37.51% 37.51%     0.53GB 37.51%  bufio.NewReaderSize
    0.11GB  7.61% 45.12%     0.11GB  7.61%  net/http.(*Request).WithContext
    0.06GB  4.08% 49.20%     0.06GB  4.08%  net/http.Header.Clone (inline)
    0.05GB  3.77% 52.97%     0.05GB  3.77%  io.ReadAll
    0.05GB  3.67% 56.64%     0.05GB  3.67%  net/textproto.MIMEHeader.Add
    0.04GB  3.08% 59.71%     0.04GB  3.08%  net/textproto.MIMEHeader.Set (inline)
    0.04GB  3.02% 62.74%     0.04GB  3.02%  github.com/sastromikus/pip_shortener/internal/repository.(*MemoryRepository).AddUserURL
    0.04GB  2.56% 65.30%     0.04GB  2.56%  github.com/sirupsen/logrus.(*Entry).Dup
    0.03GB  2.39% 67.68%     0.03GB  2.39%  crypto/internal/fips140/sha256.New (inline)
    0.03GB  2.18% 69.86%     0.06GB  4.36%  net/http.readRequest
    0.03GB  2.11% 71.97%     0.03GB  2.11%  unicode/utf16.Encode
    0.03GB  2.04% 74.01%     0.06GB  4.43%  crypto/internal/fips140/hmac.New[go.shape.interface { BlockSize int; Reset; Size int; Sum []uint8; Write  }]
    0.03GB  2.01% 76.01%     0.10GB  6.81%  github.com/sastromikus/pip_shortener/internal/handler/middleware.signCookie
    0.03GB  1.97% 77.98%     0.12GB  8.78%  github.com/sastromikus/pip_shortener/internal/handler/middleware.buildCookie
    0.03GB  1.83% 79.82%     0.03GB  1.83%  net/http/httptest.NewRecorder
    0.02GB  1.73% 81.55%     0.02GB  1.73%  strings.(*Builder).grow
    0.02GB  1.59% 83.14%     0.21GB 14.90%  github.com/sastromikus/pip_shortener/internal/handler.handleShorten
    0.02GB  1.51% 84.64%     0.02GB  1.51%  github.com/sastromikus/pip_shortener/internal/repository.(*MemoryRepository).Put
    0.02GB  1.45% 86.09%     0.03GB  1.97%  net/http/httptest.(*ResponseRecorder).Result
    0.02GB  1.35% 87.44%     0.04GB  2.56%  github.com/sirupsen/logrus.(*TextFormatter).Format
    0.02GB  1.31% 88.76%     0.02GB  1.31%  github.com/sirupsen/logrus.(*Entry).WithFields
    0.01GB  1.00% 89.76%     0.01GB  1.00%  net/url.parse
    0.01GB   0.8% 90.55%     0.01GB   0.8%  context.WithValue
    0.01GB  0.73% 91.28%     0.01GB  0.73%  runtime.allocm
    0.01GB  0.73% 92.01%     0.01GB  0.73%  github.com/sirupsen/logrus.(*Logger).releaseEntry
```
