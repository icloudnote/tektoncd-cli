## tkn pipeline

Parent command of the `pipeline` command group

### Synopsis

Parent command of the `pipeline` command group

### Aliases

```
pipeline, p, pipelines
```

### Available Commands

```
describe    Describes a pipeline in a namespace
list        Lists pipelines in a namespace
start       Start pipelines by creating a pipelinerun in a namespace
```

### Options

```
-h, --help                help for pipeline
-k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
-n, --namespace string    namespace to use (default: from $KUBECONFIG)
```

### SEE ALSO

* [tkn pipeline list](tkn_pipeline_list.md)	 - Lists all `pipelines` in a given namespace.
* [tkn pipeline describe](tkn_pipeline_describe.md)	 - Describe given `pipeline`.
* [tkn pipeline start](tkn_pipeline_start.md)	 - Start given `pipeline`.