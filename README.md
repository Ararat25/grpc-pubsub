# grpc-pubsub
Это сервис подписок, работающий по gRPC и использующий пакет subpub.

## ▶️ Инструкция по запуску

Для запуска программы и тестов используется `Makefile`. Убедитесь, что у вас установлен `make`.

### Конфигурация

Конфигурация задается через YAML-файл `config.yml`:

```yaml
server:
  port: 8080
  timeout: 5s
```

Путь к файлу конфигурации определяется приоритетно:

1. С помощью CONFIG_PATH в .env

2. По умолчанию ../config.yml

Для возможности задать путь к файлу конфигурации в файле .env необходимо:

1. Создать в корневой папке проекта файл .env
2. Прописать в нем переменную CONFIG_PATH и её значение

#### Пример .env файла

```
CONFIG_PATH = "../config.yml"
```

### Сборка и запуск программы

Для сборки программы выполните команду:

```sh
make build
```

Для запуска сервера, в зависимоти от системы и оболочки запуска, выполните команду:
- Windows
  ```shell
  cd cmd && main.exe
  ```
- Linux (или MINGW)
  ```shell
  cd cmd && ./main
  ```
  
### Тесты

Для запуска тестов выполните команду:

```sh
make test
```

### Генерация gRPC-кода

Для генерации gRPC-кода выполните команду:

```sh
make generate
```

## 1. Описание пакета subpub

Пакет реализует простую шину событий, работающую по принципу Publisher-Subscriber.

Требования к шине:
- На один subject может подписываться (и отписываться) множество подписчиков.
- Один медленный подписчик не должен тормозить остальных.
- Нельзя терять порядок порядок сообщений (FIFO очередь).
- Метод Close должен учитывать переданный контекст. Если он отменен - выходим сразу, работающие хендлеры
оставляем работать.
- Горутины (если они будут) течь не должны.

### Использование пакета
- **Создание**

    ```go
    sp := subpub.NewSubPub()
    ```

- **Подписка**

    ```go
    sub, _ := sp.Subscribe("order.created", func(msg interface{}) {
        fmt.Println("Получено сообщение:", msg)
    })
    ```

  - При подписке создаётся subscription с темой и callback-функцией.

  - Подписка сохраняется в subscribers[subject].

  - Возвращается объект Subscription с методом Unsubscribe.

- **Публикация**
    
    ```go
    sp.Publish("order.created", "Новый заказ №123")
    ```

  - Сообщение передаётся всем подписчикам на данную тему.

  - Для каждого подписчика запускается горутина с вызовом cb(msg).

  - Если subPub закрыт, возвращается ошибка.

- **Отписка**

    ```go
    sub.Unsubscribe()
    ```

  - Удаляет подписку из subscribers[subject]

- **Закрытие**

    ```go
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()
    sp.Close(ctx)
    ```

### Тестирование

- В пакете реализованы unit-тесты

## 2. Описание сервиса

Сервис предоставляет возможность подписаться на события по ключу и опубликовать события по ключу для всех
подписчиков.

### Protobuf-схема gRPC сервиса:

```proto
syntax = "proto3";

package pubsub;

option go_package = "github.com/Ararat25/grpc-pubsub;pubsub";

import "google/protobuf/empty.proto";

service PubSub {
  // Подписка (сервер отправляет поток событий)
  rpc Subscribe(SubscribeRequest) returns (stream Event);
  // Публикация (классический запрос-ответ)
  rpc Publish(PublishRequest) returns (google.protobuf.Empty);
}

message SubscribeRequest {
  string key = 1;
}

message PublishRequest {
  string key = 1;
  string data = 2;
}

message Event {
  string data = 1;
}

```

### Метод Subscribe
- Клиент отправляет запрос на подписку с определенным ключом
- Сервер подписывает клиента на тему по ключу
- При разрыве соединения подписка удаляется
#### Пример запроса
```
{
    "key": "order.created"
}
```
#### Пример ответа
```
{}
```
### Метод Publish
- Клиент отправляет запрос на сервер с информацией для публикации (data) и темой (key) 
- Сервер отправляет полученное сообщение всем подписчикам в данной теме
#### Пример запроса
```
{
    "data": "Новый заказ №123",
    "key": "order.created"
}
```
#### Пример ответа
```
{}
```

## Дополнительная информация

1. Добавлено логирование одиночных и потоковых gRPC-запросов через Interceptor
2. Реализован паттерн graceful shutdown для завершения работы сервера
3. Реализован паттерн dependency injection: srv := api.NewServer(subpubService)
4. Написаны unit-тесты для сервера
