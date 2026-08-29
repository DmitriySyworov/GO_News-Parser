{{- define "common-lib.generateAllConfig" -}}
{{- range $configName, $configData := .Values.configs }}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ $configName }}
data:
{{ $configData | toYaml | nindent 2 }}
---
{{- end }}
{{- end -}}