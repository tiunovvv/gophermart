# go-musthave-diploma-tpl

Шаблон репозитория для индивидуального дипломного проекта курса «Go-разработчик»

# Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без
   префикса `https://`) для создания модуля

# Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m master template https://github.com/yandex-praktikum/go-musthave-diploma-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/master .github
```

Затем добавьте полученные изменения в свой репозиторий.

Запуск:
```
go run cmd/gophermart/main.go -d="postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
```
```
cmd/accrual/accrual_linux_amd64 -a="localhost:8000" -d="postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
```

Тест:
```
gophermarttest -test.v -test.run=^TestGophermart$ \
            -gophermart-binary-path=cmd/gophermart/gophermart \
            -gophermart-host=localhost \
            -gophermart-port=8080 \
            -gophermart-database-uri="postgresql://postgres:postgres@localhost/praktikum?sslmode=disable" \
            -accrual-binary-path=cmd/accrual/accrual_linux_amd64 \
            -accrual-host=localhost \
            -accrual-port=8000 \
            -accrual-database-uri="postgresql://postgres:postgres@localhost/praktikum?sslmode=disable"
```