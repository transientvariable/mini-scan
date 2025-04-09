package miniscan

const (
	// Opensearch configuration path.
	//
	// String: <root>.repository.opensearch
	Opensearch = "config.repository.opensearch"

	// Addresses configuration path.
	//
	// String: <root>.repository.opensearch.addresses
	Addresses = Opensearch + ".addresses"

	// Username configuration path.
	//
	// String: <root>.repository.opensearch.username
	Username = Opensearch + ".username"

	// Password configuration path.
	//
	// String: <root>.repository.opensearch.password
	Password = Opensearch + ".password"

	// Mapping configuration path.
	//
	// String: <root>.repository.opensearch.mapping
	Mapping = Opensearch + ".mapping"

	// MappingCreate configuration path.
	//
	// String: <root>.repository.opensearch.mapping.create
	MappingCreate = Mapping + ".create"

	// MappingTemplatePath configuration path.
	//
	// String: <root>.repository.opensearch.mapping.templatePath
	MappingTemplatePath = Mapping + ".templatePath"

	// MappingIndicesPath configuration path.
	//
	// String: <root>.repository.opensearch.mapping.indicesPath
	MappingIndicesPath = Mapping + ".indicesPath"

	// Retry configuration path.
	//
	// String: <root>.repository.opensearch.retry
	Retry = Opensearch + ".retry"

	// RetryEnable configuration path.
	//
	// String: <root>.repository.opensearch.retry.enable
	RetryEnable = Retry + ".enable"

	// RetryStatus configuration path.
	//
	// String: <root>.repository.opensearch.retry.status
	RetryStatus = Retry + ".status"

	// RetryMax configuration path.
	//
	// String: <root>.repository.opensearch.retry.max
	RetryMax = Retry + ".max"

	// Bulk configuration path.
	//
	// String: <root>.repository.opensearch.bulk
	Bulk = Opensearch + ".bulk"

	// BulkFlush configuration path.
	//
	// String: <root>.repository.opensearch.bulk.flush
	BulkFlush = Bulk + ".flush"

	// BulkFlushSize configuration path.
	//
	// String: <root>.repository.opensearch.bulk.flush.size
	BulkFlushSize = BulkFlush + ".size"

	// BulkFlushInterval configuration path.
	//
	// String: <root>.repository.opensearch.bulk.flush.interval
	BulkFlushInterval = BulkFlush + ".interval"

	// BulkStatsEnable configuration path.
	//
	// String: <root>.repository.opensearch.bulk.statsEnable
	BulkStatsEnable = Bulk + ".statsEnable"

	// BulkWorkers configuration path.
	//
	// String: <root>.repository.opensearch.bulk.workers
	BulkWorkers = Bulk + ".workers"
)
