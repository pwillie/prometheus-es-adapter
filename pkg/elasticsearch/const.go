package elasticsearch

const sampleType = "sample"

const indexCreate = `{
	"aliases": {
		"{{.Alias}}": {}
	}
}`

const indexTemplate = `{
	"index_patterns": ["{{.Alias}}-*"],
	"settings": {
		"number_of_shards": {{.Shards}},
		"number_of_replicas": {{.Replicas}}
	},
	"mappings": {
		"sample": {
			"_source": {
				"enabled": true
			},
			"properties": {
				"timestamp": {
					"type": "date",
					"format": "strict_date_optional_time||epoch_millis"
				},
				"value": {
					"type": "double"
				}
			},
			"dynamic_templates": [
				{
					"strings": {
						"match_mapping_type": "string",
						"path_match": "label.*",
						"mapping": {
							"type": "keyword"
						}
					}
				}
			]
		}
	}
}`
