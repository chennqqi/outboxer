=== RUN   TestOutboxMessage
=== PAUSE TestOutboxMessage
=== RUN   TestDynamicValues_Scan_InvalidType
--- PASS: TestDynamicValues_Scan_InvalidType (0.00s)
=== RUN   TestDynamicValues_Scan_InvalidSrc
--- PASS: TestDynamicValues_Scan_InvalidSrc (0.00s)
=== RUN   TestOutboxer_Send
started to listen for new messages
sending message...
waiting for successfully sent messages...
message received
--- PASS: TestOutboxer_Send (1.02s)
=== RUN   TestOutboxer_SendWithinTx
started to listen for new messages
waiting for successfully sent messages...
message received
--- PASS: TestOutboxer_SendWithinTx (1.00s)
=== RUN   TestOutboxer_WithWrongParams
--- PASS: TestOutboxer_WithWrongParams (0.00s)
=== CONT  TestOutboxMessage
=== RUN   TestOutboxMessage/check_if_DynamicValues_Scan_works_for_message
=== RUN   TestOutboxMessage/check_if_DynamicValues_Value_works_for_message
--- PASS: TestOutboxMessage (0.00s)
    --- PASS: TestOutboxMessage/check_if_DynamicValues_Scan_works_for_message (0.00s)
    --- PASS: TestOutboxMessage/check_if_DynamicValues_Value_works_for_message (0.00s)
PASS
ok  	github.com/italolelis/outboxer	2.588s
=== RUN   TestNewAMQP
--- PASS: TestNewAMQP (0.00s)
=== RUN   TestAMQP_ParseOptions_Defaults
--- PASS: TestAMQP_ParseOptions_Defaults (0.00s)
=== RUN   TestAMQP_ParseOptions_AllOptions
--- PASS: TestAMQP_ParseOptions_AllOptions (0.00s)
=== RUN   TestAMQP_ParseOptions_PartialOptions
--- PASS: TestAMQP_ParseOptions_PartialOptions (0.00s)
=== RUN   TestAMQP_ParseOptions_TypeAssertions
--- PASS: TestAMQP_ParseOptions_TypeAssertions (0.00s)
=== RUN   TestAMQP_ParseOptions_BooleanValues
=== RUN   TestAMQP_ParseOptions_BooleanValues/all_true
=== RUN   TestAMQP_ParseOptions_BooleanValues/all_false
--- PASS: TestAMQP_ParseOptions_BooleanValues (0.00s)
    --- PASS: TestAMQP_ParseOptions_BooleanValues/all_true (0.00s)
    --- PASS: TestAMQP_ParseOptions_BooleanValues/all_false (0.00s)
=== RUN   TestAMQP_Constants
--- PASS: TestAMQP_Constants (0.00s)
PASS
ok  	github.com/italolelis/outboxer/es/amqp	0.534s
=== RUN   TestKinesis_EventStream
--- PASS: TestKinesis_EventStream (0.02s)
PASS
ok  	github.com/italolelis/outboxer/es/kinesis	(cached)
=== RUN   TestPublishMessages
=== RUN   TestPublishMessages/sending_a_message
=== RUN   TestPublishMessages/sending_a_message_to_non_existent_topic
--- PASS: TestPublishMessages (0.08s)
    --- PASS: TestPublishMessages/sending_a_message (0.01s)
    --- PASS: TestPublishMessages/sending_a_message_to_non_existent_topic (0.01s)
PASS
ok  	github.com/italolelis/outboxer/es/pubsub	(cached)
=== RUN   TestRabbitMQ_ParseOptions_Defaults
--- PASS: TestRabbitMQ_ParseOptions_Defaults (0.00s)
=== RUN   TestRabbitMQ_ParseOptions_PriorityTypes
=== RUN   TestRabbitMQ_ParseOptions_PriorityTypes/uint8
=== RUN   TestRabbitMQ_ParseOptions_PriorityTypes/int
=== RUN   TestRabbitMQ_ParseOptions_PriorityTypes/string
=== RUN   TestRabbitMQ_ParseOptions_PriorityTypes/invalid_int
=== RUN   TestRabbitMQ_ParseOptions_PriorityTypes/invalid_string
--- PASS: TestRabbitMQ_ParseOptions_PriorityTypes (0.00s)
    --- PASS: TestRabbitMQ_ParseOptions_PriorityTypes/uint8 (0.00s)
    --- PASS: TestRabbitMQ_ParseOptions_PriorityTypes/int (0.00s)
    --- PASS: TestRabbitMQ_ParseOptions_PriorityTypes/string (0.00s)
    --- PASS: TestRabbitMQ_ParseOptions_PriorityTypes/invalid_int (0.00s)
    --- PASS: TestRabbitMQ_ParseOptions_PriorityTypes/invalid_string (0.00s)
=== RUN   TestRabbitMQ_ParseOptions_DeliveryModeTypes
=== RUN   TestRabbitMQ_ParseOptions_DeliveryModeTypes/uint8_1
=== RUN   TestRabbitMQ_ParseOptions_DeliveryModeTypes/uint8_2
=== RUN   TestRabbitMQ_ParseOptions_DeliveryModeTypes/int_1
=== RUN   TestRabbitMQ_ParseOptions_DeliveryModeTypes/int_2
=== RUN   TestRabbitMQ_ParseOptions_DeliveryModeTypes/string_1
=== RUN   TestRabbitMQ_ParseOptions_DeliveryModeTypes/string_2
=== RUN   TestRabbitMQ_ParseOptions_DeliveryModeTypes/string_non_persistent
=== RUN   TestRabbitMQ_ParseOptions_DeliveryModeTypes/string_persistent
=== RUN   TestRabbitMQ_ParseOptions_DeliveryModeTypes/invalid
--- PASS: TestRabbitMQ_ParseOptions_DeliveryModeTypes (0.00s)
    --- PASS: TestRabbitMQ_ParseOptions_DeliveryModeTypes/uint8_1 (0.00s)
    --- PASS: TestRabbitMQ_ParseOptions_DeliveryModeTypes/uint8_2 (0.00s)
    --- PASS: TestRabbitMQ_ParseOptions_DeliveryModeTypes/int_1 (0.00s)
    --- PASS: TestRabbitMQ_ParseOptions_DeliveryModeTypes/int_2 (0.00s)
    --- PASS: TestRabbitMQ_ParseOptions_DeliveryModeTypes/string_1 (0.00s)
    --- PASS: TestRabbitMQ_ParseOptions_DeliveryModeTypes/string_2 (0.00s)
    --- PASS: TestRabbitMQ_ParseOptions_DeliveryModeTypes/string_non_persistent (0.00s)
    --- PASS: TestRabbitMQ_ParseOptions_DeliveryModeTypes/string_persistent (0.00s)
    --- PASS: TestRabbitMQ_ParseOptions_DeliveryModeTypes/invalid (0.00s)
=== RUN   TestRabbitMQ_ParseOptions_AllOptions
--- PASS: TestRabbitMQ_ParseOptions_AllOptions (0.00s)
PASS
ok  	github.com/italolelis/outboxer/es/rabbitmq	(cached)
=== RUN   TestSQS_EventStream
--- PASS: TestSQS_EventStream (0.00s)
PASS
ok  	github.com/italolelis/outboxer/es/sqs	(cached)
?   	github.com/italolelis/outboxer/examples	[no test files]
=== RUN   TestLock_Generate
--- PASS: TestLock_Generate (0.00s)
PASS
ok  	github.com/italolelis/outboxer/lock	(cached)
=== RUN   TestMongoDB_Close_Success
--- PASS: TestMongoDB_Close_Success (0.05s)
=== RUN   TestMongoDB_GetEvents_EmptyResult
--- PASS: TestMongoDB_GetEvents_EmptyResult (0.07s)
=== RUN   TestMongoDB_GetEvents_WithDispatchedAt
--- PASS: TestMongoDB_GetEvents_WithDispatchedAt (0.07s)
=== RUN   TestMongoDB_SetAsDispatched_NotFound
--- PASS: TestMongoDB_SetAsDispatched_NotFound (0.07s)
=== RUN   TestMongoDB_Remove_EmptyResult
--- PASS: TestMongoDB_Remove_EmptyResult (0.07s)
=== RUN   TestMongoDB_Remove_WithBatchSize
--- PASS: TestMongoDB_Remove_WithBatchSize (0.09s)
=== RUN   TestMongoDB_Lock_Unlock
--- PASS: TestMongoDB_Lock_Unlock (0.07s)
=== RUN   TestMongoDB_Lock_ExpiredLock
--- PASS: TestMongoDB_Lock_ExpiredLock (0.08s)
=== RUN   TestMongoDB_AddWithinTx_WithoutTransactionSupport
--- PASS: TestMongoDB_AddWithinTx_WithoutTransactionSupport (0.07s)
=== RUN   TestMongoDB_AddWithinTx_BusinessLogicError
--- PASS: TestMongoDB_AddWithinTx_BusinessLogicError (0.07s)
=== RUN   TestMongoDB_EnsureIndexes
--- PASS: TestMongoDB_EnsureIndexes (0.08s)
=== RUN   TestMongoDB_WithInstance_PingError
--- PASS: TestMongoDB_WithInstance_PingError (30.00s)
=== RUN   TestMongoExecer_ExecContext
--- PASS: TestMongoExecer_ExecContext (0.00s)
=== RUN   TestMongoDB_GetEvents_BatchSize
--- PASS: TestMongoDB_GetEvents_BatchSize (0.10s)
=== RUN   TestMongoDB_WithInstance_NoDB
--- PASS: TestMongoDB_WithInstance_NoDB (0.00s)
=== RUN   TestMongoDB_AddSuccessfully
--- PASS: TestMongoDB_AddSuccessfully (0.07s)
=== RUN   TestMongoDB_GetEventsSuccessfully
--- PASS: TestMongoDB_GetEventsSuccessfully (0.07s)
=== RUN   TestMongoDB_SetAsDispatchedSuccessfully
--- PASS: TestMongoDB_SetAsDispatchedSuccessfully (0.07s)
=== RUN   TestMongoDB_RemoveSuccessfully
--- PASS: TestMongoDB_RemoveSuccessfully (0.19s)
=== RUN   TestMongoDB_AddWithinTx
--- PASS: TestMongoDB_AddWithinTx (0.07s)
PASS
ok  	github.com/italolelis/outboxer/storage/mongodb	31.794s
=== RUN   TestMySQL_CloseSuccessfully
--- PASS: TestMySQL_CloseSuccessfully (0.00s)
=== RUN   TestMySQL_GetEventsSuccessfully
--- PASS: TestMySQL_GetEventsSuccessfully (0.00s)
=== RUN   TestMySQL_GetEventsWhenThereIsNoMessage
--- PASS: TestMySQL_GetEventsWhenThereIsNoMessage (0.00s)
=== RUN   TestMySQL_GetEventsWithAnSQLError
--- PASS: TestMySQL_GetEventsWithAnSQLError (0.00s)
=== RUN   TestMySQL_GetEventsWithAFailedScan
--- PASS: TestMySQL_GetEventsWithAFailedScan (0.00s)
=== RUN   TestMySQL_SetAsDispatchedSuccessfully
--- PASS: TestMySQL_SetAsDispatchedSuccessfully (0.00s)
=== RUN   TestMySQL_SetAsDispatchedWithSQLError
--- PASS: TestMySQL_SetAsDispatchedWithSQLError (0.00s)
=== RUN   TestMySQL_RemoveSuccessfully
--- PASS: TestMySQL_RemoveSuccessfully (0.00s)
=== RUN   TestMySQL_RemoveWithSQLError
--- PASS: TestMySQL_RemoveWithSQLError (0.00s)
=== RUN   TestMySQL_RemoveTxFails
--- PASS: TestMySQL_RemoveTxFails (0.00s)
=== RUN   TestMySQL_AddSuccessfully
--- PASS: TestMySQL_AddSuccessfully (0.00s)
=== RUN   TestMySQL_AddFails
--- PASS: TestMySQL_AddFails (0.00s)
=== RUN   TestMySQL_WithInstanceNoDB
--- PASS: TestMySQL_WithInstanceNoDB (0.00s)
=== RUN   TestMySQL_WithInstanceWithEmptyDBName
--- PASS: TestMySQL_WithInstanceWithEmptyDBName (0.00s)
=== RUN   TestMySQL_AddWithinTx
--- PASS: TestMySQL_AddWithinTx (0.00s)
PASS
ok  	github.com/italolelis/outboxer/storage/mysql	(cached)
=== RUN   TestPostgres_AddSuccessfully
--- PASS: TestPostgres_AddSuccessfully (0.00s)
=== RUN   TestPostgres_WithInstanceNoDB
--- PASS: TestPostgres_WithInstanceNoDB (0.00s)
=== RUN   TestPostgres_WithInstanceWithEmptyDBName
--- PASS: TestPostgres_WithInstanceWithEmptyDBName (0.00s)
=== RUN   TestPostgres_WithInstanceNoSchema
--- PASS: TestPostgres_WithInstanceNoSchema (0.00s)
=== RUN   TestPostgres_WithInstanceWithEmptySchemaName
--- PASS: TestPostgres_WithInstanceWithEmptySchemaName (0.00s)
=== RUN   TestPostgres_AddWithinTx
--- PASS: TestPostgres_AddWithinTx (0.00s)
PASS
ok  	github.com/italolelis/outboxer/storage/postgres	(cached)
=== RUN   TestSQLServer_WithInstance_must_return_SQLServerDataStore
--- PASS: TestSQLServer_WithInstance_must_return_SQLServerDataStore (0.00s)
=== RUN   TestSQLServer_WithInstance_should_return_error_when_no_db_selected
--- PASS: TestSQLServer_WithInstance_should_return_error_when_no_db_selected (0.00s)
=== RUN   TestSQLServer_WithInstance_should_return_error_when_no_schema_selected
--- PASS: TestSQLServer_WithInstance_should_return_error_when_no_schema_selected (0.00s)
=== RUN   TestSQLServer_should_add_message
--- PASS: TestSQLServer_should_add_message (0.00s)
=== RUN   TestSQLServer_should_add_message_with_tx
--- PASS: TestSQLServer_should_add_message_with_tx (0.00s)
=== RUN   TestSQLServer_add_message_with_tx_should_rollback_on_error
--- PASS: TestSQLServer_add_message_with_tx_should_rollback_on_error (0.00s)
=== RUN   TestSQLServer_should_get_events
--- PASS: TestSQLServer_should_get_events (0.00s)
=== RUN   TestSQLServer_should_set_as_dispatched
--- PASS: TestSQLServer_should_set_as_dispatched (0.00s)
=== RUN   TestSQLServer_should_remove_messages
--- PASS: TestSQLServer_should_remove_messages (0.00s)
=== RUN   TestRemoveMessages_Rollback
--- PASS: TestRemoveMessages_Rollback (0.00s)
=== RUN   TestRemoveMessages_Failed_Rollback
--- PASS: TestRemoveMessages_Failed_Rollback (0.00s)
PASS
ok  	github.com/italolelis/outboxer/storage/sqlserver	(cached)
