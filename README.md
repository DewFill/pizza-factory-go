# Overview

This API provides endpoints for managing pizza orders, including creating orders, adding items to orders, marking orders as done, and listing orders. Authentication is required for certain operations.

API tests can be found in file [test-requests.http](test-requests.http)

Database schema can be found in file [docker/postgres/docker-entrypoint-initdb.d/schema.sql](docker/postgres/docker-entrypoint-initdb.d/schema.sql)

App queries can be found in file [query.sql](query.sql)
## Start
```shell
make up
```

## Stop
Stop all containers without losing data
```shell
make down
```

## Remove
Delete all containers, volumes, and networks
```shell
make clear
```

# Endpoints
## Create Order
Create a new order.

#### Endpoint:
`POST /orders`

#### Headers:
* Content-Type: application/json

#### Request Body

```json
{
  "item_ids": [
    1,
    1,
    2
  ]
}
```
#### Response

* 201 Created: Returns the created order.
```json
{
  "order_id": "dd9fd3a2c8405e8",
  "items": [
    4,
    5,
    6
  ],
  "done": false
}
```
* 400 Bad Request: If the request body is invalid.
* 404 Not Found: Item not found.

## Add Items to Order
Add items to an existing order.

#### Endpoint:
`POST /orders/{order_id}/items`

#### Headers:
* Content-Type: application/json

#### Request Body

```json
[
  1,
  1,
  2
]
```
#### Response

* 200 OK
* 400 Bad Request: If the request body is invalid.
* 404 Not Found: Order or Item not found.
* 409 Conflict: Order is already done.






## Get Order
Retrieve details of a specific order by its ID.

#### Endpoint: 
`GET /orders/{order_id}`

#### Parameters:

* `order_id` (path): The ID of the order to retrieve.
#### Response:

* `200 OK`: Returns the order details.
* `404 Not Found`: If the order with the specified ID does not exist.




## Make order done (requires auth)
Changes field of existing order to 'done'.

#### Endpoint:
`POST /orders/{{order_id}}/done`

#### Headers:
* X-Auth-Key: `{{auth_key}}`

#### Response

* 200 OK
* 404 Not Found: Order not found.
* 409 Conflict: Order is already done.
* 401 Unauthorized: Auth key is not provided or does not match.




## List orders (requires auth)
List all orders

#### Endpoint:
`GET /orders/[?done=1|0]`

URL queries:
* `done` - If present, filters for done (?done=1) or not done (?done=0) orders. Omit to request all orders.
#### Headers:
* X-Auth-Key: `{{auth_key}}`

#### Response

* 200 OK
```json
[
  {
    "order_id": "544761bce741f18",
    "done": false
  },
  {
    "order_id": "dd9fd3a2c8405e8",
    "done": true
  }
]
```
* 400 Bad Request: `done` query is an invalid value.
* 401 Unauthorized: Auth key is not provided or does not match.