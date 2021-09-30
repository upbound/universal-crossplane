{{- define "agent-spec" -}}
replicas: 1
selector:
  matchLabels:
    {{- include "selectorLabelsAgent" . | nindent 8 }}
template:
  metadata:
    labels:
      {{- include "selectorLabelsAgent" . | nindent 10 }}
  spec:
    serviceAccountName: {{ template "agent-name" . }}
    {{- if .Values.imagePullSecrets }}
    imagePullSecrets:
    {{- range $index, $secret := .Values.imagePullSecrets }}
    - name: {{ $secret }}
    {{- end }}
    {{ end }}
    containers:
      - name: agent
        image: "{{ .Values.agent.image.repository }}:{{ .Values.agent.image.tag }}"
        args:
        - agent
        - --tls-cert-file
        - /etc/certs/upbound-agent/tls.crt
        - --tls-key-file
        - /etc/certs/upbound-agent/tls.key
        - --xgql-ca-bundle-file
        - /etc/certs/upbound-agent/ca.crt
        - --nats-endpoint
        - nats://{{ .Values.upbound.connectHost }}:{{ .Values.upbound.connectPort | default "443" }}
        - --upbound-api-endpoint
        - {{ .Values.upbound.apiURL }}
        {{- if .Values.agent.config.debugMode }}
        - "--debug"
        {{- end }}
        {{- range $arg := .Values.agent.config.args }}
        - {{ $arg }}
        {{- end }}
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: CONTROL_PLANE_TOKEN
          valueFrom:
            secretKeyRef:
              name: {{ .Values.upbound.controlPlane.tokenSecretName }}
              key: token
        {{- range $key, $value := .Values.agent.config.envVars }}
        - name: {{ $key | replace "." "_" }}
          value: {{ $value | quote }}
        {{- end}}
        imagePullPolicy: {{ .Values.agent.image.pullPolicy }}
        ports:
        - name: agent
          containerPort: 6443
          protocol: TCP
        resources:
          {{- toYaml .Values.agent.resources | nindent 14 }}
        readinessProbe:
          httpGet:
            scheme: HTTPS
            path: /readyz
            port: 6443
          initialDelaySeconds: 5
          timeoutSeconds: 5
          periodSeconds: 5
          failureThreshold: 3
        livenessProbe:
          httpGet:
            scheme: HTTPS
            path: /livez
            port: 6443
          initialDelaySeconds: 10
          timeoutSeconds: 5
          periodSeconds: 30
          failureThreshold: 5
        volumeMounts:
          - mountPath: /etc/certs/upbound-agent
            name: certs
            readOnly: true
    volumes:
      - name: certs
        secret:
          defaultMode: 420
          secretName: upbound-agent-tls
{{- end }}