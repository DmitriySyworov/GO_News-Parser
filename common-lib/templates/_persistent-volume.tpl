{{- define "common-lib.generateAllPVs" -}}
{{- range .Values.statefulServices }}
{{- if .isLocal }}
apiVersion: v1
kind: PersistentVolume
metadata:
  name: {{ .name }}-pv
spec:
  capacity:
    storage: {{ .storageSize }}
  volumeMode: Filesystem
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: local-storage
  local:
    path: /mnt/data
  nodeAffinity:
    required:
      nodeSelectorTerms:
        - matchExpressions:
            - key: kubernetes.io/hostname
              operator: In
              values:
                - minikube
---
{{- end }}
{{- end }}
{{- end -}}