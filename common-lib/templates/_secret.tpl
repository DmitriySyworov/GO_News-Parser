{{- define "common-lib.generateAllSecret" -}}
{{- range $secretName, $secretData := .Values.secrets }}
apiVersion: v1
kind: Secret
metadata:
  name: {{ $secretName }}
type: Opaque
stringData:
{{ toYaml $secretData | indent 4 }}
---
{{- end }}
{{- end -}}