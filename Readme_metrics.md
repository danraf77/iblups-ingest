# 📊 Sistema de Métricas SRS - Documentación para Dashboard

## 📋 Tabla de Contenidos

- [Arquitectura General](#arquitectura-general)
- [Base de Datos](#base-de-datos)
- [API Endpoints](#api-endpoints)
- [Modelos de Datos](#modelos-de-datos)
- [Queries SQL Útiles](#queries-sql-útiles)
- [Ejemplos de Integración](#ejemplos-de-integración)
- [Casos de Uso](#casos-de-uso)
- [Websockets y Real-time](#websockets-y-real-time)

---

## 🏗️ Arquitectura General

```
┌─────────────────┐
│   SRS Server    │ ← Puerto 1935 (RTMP), 1985 (API), 8080 (HTTP)
│  (1 o N servers)│
└────────┬────────┘
         │ Métricas cada 30s
         ↓
┌─────────────────┐
│  Backend Go     │ ← Puerto 3000
│  (metrics_      │
│   collector)    │
└────────┬────────┘
         │ Guarda métricas
         ↓
┌─────────────────┐
│   Supabase      │ ← PostgreSQL
│   (5 tablas +   │
│    4 vistas)    │
└────────┬────────┘
         │ Consulta
         ↓
┌─────────────────┐
│  Dashboard      │ ← Next.js / React
│  (Tu Frontend)  │
└─────────────────┘
```

**Flujo de datos:**

1. SRS Server genera métricas cada 30 segundos
2. Backend Go las recopila vía HTTP API
3. Se guardan en Supabase con `server_id` único
4. Dashboard consulta via REST o Supabase Client
5. Actualización en tiempo real opcional con Supabase Realtime

---

## 🗄️ Base de Datos

### Tablas Principales

#### 1. `iblups_srs_servers` - Registro de Servidores

**Propósito:** Catálogo de todos los servidores SRS en la infraestructura.

| Campo         | Tipo         | Descripción                      | Ejemplo                  |
| ------------- | ------------ | -------------------------------- | ------------------------ |
| `id`          | BIGSERIAL    | ID autoincremental               | 1                        |
| `server_id`   | VARCHAR(100) | Identificador único del servidor | `srs-paris-01`           |
| `server_ip`   | VARCHAR(50)  | IP pública del servidor          | `51.210.109.197`         |
| `server_name` | VARCHAR(255) | Nombre descriptivo               | `SRS Paris Principal`    |
| `location`    | VARCHAR(100) | Ubicación física                 | `Paris, France`          |
| `is_active`   | BOOLEAN      | Estado del servidor              | `true`                   |
| `last_seen`   | TIMESTAMPTZ  | Última métrica recibida          | `2026-02-06 10:30:00+00` |
| `created_at`  | TIMESTAMPTZ  | Fecha de registro                | `2026-02-01 08:00:00+00` |
| `metadata`    | JSONB        | Datos adicionales flexibles      | `{"provider":"OVH"}`     |

**Query de ejemplo:**

```sql
SELECT
    server_id,
    server_name,
    location,
    is_active,
    last_seen,
    EXTRACT(EPOCH FROM (NOW() - last_seen)) as seconds_offline
FROM iblups_srs_servers
ORDER BY is_active DESC, last_seen DESC;
```

---

#### 2. `iblups_server_metrics` - Métricas del Servidor

**Propósito:** Historial de CPU, RAM y conexiones por servidor.

**Frecuencia:** Cada 30 segundos por servidor.

| Campo               | Tipo         | Descripción            | Rango típico     |
| ------------------- | ------------ | ---------------------- | ---------------- |
| `id`                | BIGSERIAL    | ID autoincremental     | -                |
| `timestamp`         | TIMESTAMPTZ  | Momento de la métrica  | -                |
| `server_id`         | VARCHAR(100) | ID del servidor        | `srs-paris-01`   |
| `server_ip`         | VARCHAR(50)  | IP del servidor        | `51.210.109.197` |
| `cpu_percent`       | DECIMAL(5,2) | Uso de CPU             | 0.00 - 100.00    |
| `memory_mb`         | BIGINT       | Memoria usada en MB    | 24 - 2048+       |
| `total_streams`     | INTEGER      | Streams activos        | 0 - 1000+        |
| `total_connections` | INTEGER      | Conexiones totales     | 0 - 5000+        |
| `publishers`        | INTEGER      | Clientes publicando    | 0 - 1000+        |
| `players`           | INTEGER      | Clientes reproduciendo | 0 - 4000+        |

**Query de ejemplo - CPU promedio última hora:**

```sql
SELECT
    server_id,
    ROUND(AVG(cpu_percent)::numeric, 2) as avg_cpu,
    MAX(cpu_percent) as max_cpu,
    ROUND(AVG(memory_mb)::numeric, 0) as avg_memory
FROM iblups_server_metrics
WHERE timestamp >= NOW() - INTERVAL '1 hour'
GROUP BY server_id;
```

**Query de ejemplo - Serie temporal (gráfico):**

```sql
SELECT
    DATE_TRUNC('minute', timestamp) as time,
    server_id,
    AVG(cpu_percent) as cpu,
    AVG(memory_mb) as memory,
    AVG(total_connections) as connections
FROM iblups_server_metrics
WHERE timestamp >= NOW() - INTERVAL '6 hours'
GROUP BY DATE_TRUNC('minute', timestamp), server_id
ORDER BY time ASC;
```

---

#### 3. `iblups_stream_metrics` - Métricas de Streams

**Propósito:** Métricas individuales de cada stream activo.

**Frecuencia:** Cada 30 segundos mientras el stream esté activo.

| Campo           | Tipo         | Descripción                   | Ejemplo               |
| --------------- | ------------ | ----------------------------- | --------------------- |
| `id`            | BIGSERIAL    | ID autoincremental            | -                     |
| `timestamp`     | TIMESTAMPTZ  | Momento de la métrica         | -                     |
| `server_id`     | VARCHAR(100) | Servidor donde está el stream | `srs-paris-01`        |
| `server_ip`     | VARCHAR(50)  | IP del servidor               | `51.210.109.197`      |
| `stream_id`     | VARCHAR(100) | ID interno del stream         | `vid-58z524x`         |
| `stream_name`   | VARCHAR(255) | Nombre del stream             | `3e51936df37dd48a...` |
| `app`           | VARCHAR(100) | Aplicación RTMP               | `live`                |
| `clients`       | INTEGER      | Espectadores actuales         | 0 - 1000+             |
| `recv_kbps`     | INTEGER      | Bitrate de entrada            | 0 - 10000+            |
| `send_kbps`     | INTEGER      | Bitrate de salida             | 0 - 50000+            |
| `is_publishing` | BOOLEAN      | Stream está publicando        | `true` / `false`      |
| `video_codec`   | VARCHAR(50)  | Codec de video                | `H264`, `H265`        |
| `resolution`    | VARCHAR(20)  | Resolución                    | `1920x1080`           |

**Query de ejemplo - Top 10 streams por espectadores:**

```sql
SELECT
    server_id,
    stream_name,
    MAX(clients) as max_viewers,
    ROUND(AVG(recv_kbps)::numeric, 0) as avg_bitrate_in,
    COUNT(*) as total_metrics
FROM iblups_stream_metrics
WHERE timestamp >= NOW() - INTERVAL '1 hour'
    AND is_publishing = true
GROUP BY server_id, stream_name
ORDER BY max_viewers DESC
LIMIT 10;
```

**Query de ejemplo - Distribución de resoluciones:**

```sql
SELECT
    resolution,
    COUNT(DISTINCT stream_name) as total_streams,
    SUM(clients) as total_viewers
FROM iblups_stream_metrics
WHERE timestamp >= NOW() - INTERVAL '5 minutes'
    AND is_publishing = true
GROUP BY resolution
ORDER BY total_viewers DESC;
```

---

#### 4. `iblups_client_connections` - Histórico de Conexiones

**Propósito:** Registro detallado de cada conexión de cliente (publishers y players).

**Nota:** Esta tabla se llena cuando los clientes se **desconectan**, conteniendo el resumen de toda la sesión.

| Campo              | Tipo          | Descripción              | Ejemplo                     |
| ------------------ | ------------- | ------------------------ | --------------------------- |
| `id`               | BIGSERIAL     | ID autoincremental       | -                           |
| `timestamp`        | TIMESTAMPTZ   | Momento del registro     | -                           |
| `server_id`        | VARCHAR(100)  | Servidor usado           | `srs-paris-01`              |
| `server_ip`        | VARCHAR(50)   | IP del servidor          | `51.210.109.197`            |
| `client_ip`        | VARCHAR(50)   | IP del cliente           | `190.237.26.247`            |
| `client_type`      | VARCHAR(50)   | Tipo de cliente          | `fmle-publish`, `rtmp-play` |
| `stream_name`      | VARCHAR(255)  | Stream al que se conectó | `3e51936...`                |
| `app`              | VARCHAR(100)  | Aplicación RTMP          | `live`                      |
| `connected_at`     | TIMESTAMPTZ   | Inicio de conexión       | `2026-02-06 10:00:00`       |
| `disconnected_at`  | TIMESTAMPTZ   | Fin de conexión          | `2026-02-06 11:30:00`       |
| `total_send_mb`    | DECIMAL(12,2) | MB enviados              | 1234.56                     |
| `total_recv_mb`    | DECIMAL(12,2) | MB recibidos             | 5678.90                     |
| `duration_seconds` | INTEGER       | Duración total           | 5400 (90 min)               |

**Query de ejemplo - Duración promedio de sesiones:**

```sql
SELECT
    server_id,
    client_type,
    COUNT(*) as total_sessions,
    ROUND(AVG(duration_seconds)::numeric / 60, 2) as avg_duration_minutes,
    ROUND(AVG(total_recv_mb)::numeric, 2) as avg_mb_consumed
FROM iblups_client_connections
WHERE timestamp >= NOW() - INTERVAL '24 hours'
GROUP BY server_id, client_type
ORDER BY total_sessions DESC;
```

---

#### 5. `iblups_system_events` - Eventos y Alertas

**Propósito:** Log de eventos importantes y alertas del sistema.

| Campo        | Tipo         | Descripción            | Valores posibles                       |
| ------------ | ------------ | ---------------------- | -------------------------------------- |
| `id`         | BIGSERIAL    | ID autoincremental     | -                                      |
| `timestamp`  | TIMESTAMPTZ  | Momento del evento     | -                                      |
| `server_id`  | VARCHAR(100) | Servidor (nullable)    | `srs-paris-01`                         |
| `server_ip`  | VARCHAR(50)  | IP (nullable)          | `51.210.109.197`                       |
| `event_type` | VARCHAR(50)  | Tipo de evento         | Ver tabla abajo                        |
| `severity`   | VARCHAR(20)  | Nivel de severidad     | `info`, `warning`, `error`, `critical` |
| `message`    | TEXT         | Descripción del evento | -                                      |
| `metadata`   | JSONB        | Datos adicionales      | `{"cpu": 92.5}`                        |

**Tipos de eventos (`event_type`):**

- `high_cpu` - CPU > 80%
- `critical_cpu` - CPU > 90%
- `high_memory` - Memoria alta
- `stream_started` - Stream inició
- `stream_ended` - Stream terminó
- `server_offline` - Servidor no responde
- `server_online` - Servidor volvió online

**Query de ejemplo - Alertas críticas últimas 24h:**

```sql
SELECT
    timestamp,
    server_id,
    event_type,
    severity,
    message
FROM iblups_system_events
WHERE severity IN ('error', 'critical')
    AND timestamp >= NOW() - INTERVAL '24 hours'
ORDER BY timestamp DESC
LIMIT 50;
```

---

### Vistas SQL Preconstruidas

#### 1. `iblups_servers_status` - Estado Actual de Servidores

Vista consolidada con información en tiempo real de cada servidor.

```sql
SELECT * FROM iblups_servers_status;
```

**Columnas:**

- `server_id` - ID del servidor
- `server_ip` - IP del servidor
- `server_name` - Nombre del servidor
- `location` - Ubicación
- `is_active` - Estado online/offline
- `last_seen` - Última métrica
- `seconds_since_last_seen` - Segundos sin respuesta
- `current_streams` - Streams actuales
- `current_connections` - Conexiones actuales
- `current_cpu` - CPU actual
- `current_memory` - Memoria actual

**Uso en Dashboard:**

```typescript
// Tabla de estado de servidores
const { data: servers } = await supabase
  .from("iblups_servers_status")
  .select("*")
  .order("is_active", { ascending: false });
```

---

#### 2. `iblups_stats_by_server_24h` - Estadísticas por Hora

Métricas agregadas por hora de cada servidor (últimas 24h).

```sql
SELECT * FROM iblups_stats_by_server_24h
WHERE server_id = 'srs-paris-01'
ORDER BY hour DESC;
```

**Columnas:**

- `server_id`, `server_ip`
- `hour` - Hora truncada
- `avg_cpu`, `max_cpu` - CPU promedio y máxima
- `avg_memory_mb`, `max_memory_mb` - Memoria
- `avg_streams`, `avg_connections` - Streams y conexiones promedio

**Uso en Dashboard:**

```typescript
// Gráfico de CPU últimas 24h
const { data: cpuHistory } = await supabase
  .from("iblups_stats_by_server_24h")
  .select("hour, server_id, avg_cpu, max_cpu")
  .gte("hour", new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString());
```

---

#### 3. `iblups_top_streams_by_server` - Top Streams por Servidor

Los streams más populares por servidor (última hora).

```sql
SELECT * FROM iblups_top_streams_by_server
WHERE server_id = 'srs-paris-01'
LIMIT 10;
```

**Columnas:**

- `server_id`, `server_ip`
- `stream_name`, `app`
- `avg_clients` - Espectadores promedio
- `avg_recv_kbps`, `avg_send_kbps` - Bitrate promedio
- `total_metrics` - Número de muestras

---

#### 4. `iblups_load_distribution` - Distribución de Carga

Carga actual distribuida entre servidores (últimos 5 minutos).

```sql
SELECT * FROM iblups_load_distribution
ORDER BY total_clients DESC;
```

**Columnas:**

- `server_id`, `server_ip`
- `active_streams` - Streams activos
- `total_clients` - Total de espectadores
- `avg_recv_kbps`, `avg_send_kbps` - Bitrate promedio

**Uso en Dashboard:**

```typescript
// Gráfico circular de distribución de carga
const { data: distribution } = await supabase
  .from("iblups_load_distribution")
  .select("*");

// Calcular porcentaje de carga por servidor
const totalClients = distribution.reduce((sum, s) => sum + s.total_clients, 0);
const chartData = distribution.map((s) => ({
  name: s.server_id,
  value: s.total_clients,
  percentage: ((s.total_clients / totalClients) * 100).toFixed(2),
}));
```

---

## 🔌 API Endpoints (Backend Go)

**Base URL:** `http://51.210.109.197:3000/api/v1/`

### 1. `/stats` - Estadísticas Generales

**Método:** `GET`

**Descripción:** Retorna estadísticas completas del servidor SRS actual.

**Response:**

```json
{
  "server": {
    "uptime": 1738839045,
    "connections": 15,
    "total_streams": 3,
    "version": "6.0.184"
  },
  "streams": [
    {
      "id": "vid-58z524x",
      "name": "3e51936df37dd48a704e7e3c8dfd1e",
      "app": "live",
      "clients": 12,
      "recv_kbps": 2500,
      "send_kbps": 30000,
      "is_publish": true,
      "video_codec": "H264",
      "width": 1280,
      "height": 720
    }
  ],
  "resources": {
    "cpu": 45.5,
    "memory": 128
  }
}
```

**Ejemplo de uso:**

```typescript
const response = await fetch("http://51.210.109.197:3000/api/v1/stats");
const stats = await response.json();

console.log(`CPU: ${stats.resources.cpu}%`);
console.log(`Streams activos: ${stats.server.total_streams}`);
```

---

### 2. `/clients` - Clientes Conectados

**Método:** `GET`

**Descripción:** Lista de todos los clientes conectados actualmente.

**Response:**

```json
{
  "total": 15,
  "clients": [
    {
      "id": "435585qo",
      "ip": "190.237.26.247",
      "type": "fmle-publish",
      "stream": "3e51936df37dd48a...",
      "app": "live",
      "alive": 300,
      "send_bytes": 125829120,
      "recv_bytes": 0
    },
    {
      "id": "sr13a29n",
      "ip": "172.18.0.4",
      "type": "rtmp-play",
      "stream": "3e51936df37dd48a...",
      "app": "live",
      "alive": 120,
      "send_bytes": 0,
      "recv_bytes": 45678900
    }
  ]
}
```

**Ejemplo de uso:**

```typescript
const response = await fetch("http://51.210.109.197:3000/api/v1/clients");
const data = await response.json();

const publishers = data.clients.filter((c) => c.type.includes("publish"));
const players = data.clients.filter((c) => !c.type.includes("publish"));

console.log(`Publishers: ${publishers.length}`);
console.log(`Players: ${players.length}`);
```

---

### 3. `/performance` - Métricas de Rendimiento

**Método:** `GET`

**Descripción:** Métricas de rendimiento simplificadas.

**Response:**

```json
{
  "cpu": 45.5,
  "memory_mb": 128,
  "total_packets": 0,
  "total_frames": 0,
  "free_objects": 0,
  "connections": 15
}
```

---

### 4. `/summary` - Resumen del Servidor

**Método:** `GET`

**Descripción:** Resumen ejecutivo del servidor.

**Response:**

```json
{
  "version": "6.0.184",
  "pid": 1,
  "uptime": 1738839045,
  "publishers": 3,
  "players": 12,
  "total_clients": 15
}
```

---

## 📊 Queries SQL Útiles para Dashboards

### 1. Dashboard Principal - KPIs en Tiempo Real

```sql
-- Total de servidores activos
SELECT COUNT(*) as active_servers
FROM iblups_srs_servers
WHERE is_active = true;

-- Total de streams activos (todos los servidores)
SELECT SUM(total_streams) as total_streams
FROM (
    SELECT DISTINCT ON (server_id) total_streams
    FROM iblups_server_metrics
    ORDER BY server_id, timestamp DESC
) as latest;

-- Total de espectadores (todos los servidores)
SELECT SUM(players) as total_viewers
FROM (
    SELECT DISTINCT ON (server_id) players
    FROM iblups_server_metrics
    ORDER BY server_id, timestamp DESC
) as latest;

-- CPU promedio de todos los servidores
SELECT ROUND(AVG(cpu_percent)::numeric, 2) as avg_cpu
FROM (
    SELECT DISTINCT ON (server_id) cpu_percent
    FROM iblups_server_metrics
    WHERE timestamp >= NOW() - INTERVAL '5 minutes'
    ORDER BY server_id, timestamp DESC
) as latest;
```

---

### 2. Gráfico de Serie Temporal - CPU por Servidor

```sql
SELECT
    DATE_TRUNC('minute', timestamp) as time,
    server_id,
    ROUND(AVG(cpu_percent)::numeric, 2) as cpu
FROM iblups_server_metrics
WHERE timestamp >= NOW() - INTERVAL '6 hours'
GROUP BY DATE_TRUNC('minute', timestamp), server_id
ORDER BY time ASC, server_id;
```

**Resultado esperado:**

```
time                | server_id      | cpu
--------------------|----------------|------
2026-02-06 10:00:00 | srs-paris-01   | 45.50
2026-02-06 10:01:00 | srs-paris-01   | 46.20
2026-02-06 10:00:00 | srs-miami-01   | 32.10
...
```

**Uso en Chart.js o Recharts:**

```typescript
const { data } = await supabase.rpc("get_cpu_timeseries", {
  interval_hours: 6,
});

const chartData = {
  labels: [...new Set(data.map((d) => d.time))],
  datasets: [...new Set(data.map((d) => d.server_id))].map((serverId) => ({
    label: serverId,
    data: data.filter((d) => d.server_id === serverId).map((d) => d.cpu),
  })),
};
```

---

### 3. Tabla de Streams Activos con Detalles

```sql
SELECT
    sm.server_id,
    sm.stream_name,
    sm.resolution,
    sm.video_codec,
    sm.clients as viewers,
    sm.recv_kbps as bitrate_in,
    sm.send_kbps as bitrate_out,
    sm.timestamp as last_update,
    EXTRACT(EPOCH FROM (NOW() - sm.timestamp)) as seconds_ago
FROM (
    SELECT DISTINCT ON (server_id, stream_name) *
    FROM iblups_stream_metrics
    WHERE is_publishing = true
    ORDER BY server_id, stream_name, timestamp DESC
) sm
ORDER BY sm.clients DESC
LIMIT 20;
```

---

### 4. Alertas y Eventos Recientes

```sql
SELECT
    e.timestamp,
    s.server_name,
    e.event_type,
    e.severity,
    e.message,
    e.metadata
FROM iblups_system_events e
LEFT JOIN iblups_srs_servers s ON e.server_id = s.server_id
WHERE e.timestamp >= NOW() - INTERVAL '24 hours'
    AND e.severity IN ('warning', 'error', 'critical')
ORDER BY e.timestamp DESC
LIMIT 50;
```

---

### 5. Comparativa de Servidores

```sql
SELECT
    s.server_id,
    s.server_name,
    s.location,
    s.is_active,
    COALESCE(m.current_cpu, 0) as cpu,
    COALESCE(m.current_memory, 0) as memory_mb,
    COALESCE(m.current_streams, 0) as streams,
    COALESCE(m.current_connections, 0) as connections,
    COALESCE(l.total_clients, 0) as viewers
FROM iblups_srs_servers s
LEFT JOIN iblups_servers_status m ON s.server_id = m.server_id
LEFT JOIN iblups_load_distribution l ON s.server_id = l.server_id
ORDER BY s.is_active DESC, viewers DESC;
```

---

### 6. Histórico de Uso por Día (últimos 30 días)

```sql
SELECT
    DATE(timestamp) as date,
    server_id,
    ROUND(AVG(cpu_percent)::numeric, 2) as avg_cpu,
    ROUND(AVG(memory_mb)::numeric, 0) as avg_memory,
    ROUND(AVG(total_streams)::numeric, 0) as avg_streams,
    MAX(total_connections) as max_connections
FROM iblups_server_metrics
WHERE timestamp >= NOW() - INTERVAL '30 days'
GROUP BY DATE(timestamp), server_id
ORDER BY date DESC, server_id;
```

---

## 🎨 Ejemplos de Integración (Next.js + Supabase)

### Setup inicial

```bash
npm install @supabase/supabase-js
```

```typescript
// lib/supabase.ts
import { createClient } from "@supabase/supabase-js";

export const supabase = createClient(
  process.env.NEXT_PUBLIC_SUPABASE_URL!,
  process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
);
```

---

### Componente: Tabla de Servidores

```typescript
// components/ServersTable.tsx
'use client';

import { useEffect, useState } from 'react';
import { supabase } from '@/lib/supabase';

interface ServerStatus {
  server_id: string;
  server_name: string;
  location: string;
  is_active: boolean;
  current_cpu: number;
  current_memory: number;
  current_streams: number;
  current_connections: number;
  seconds_since_last_seen: number;
}

export default function ServersTable() {
  const [servers, setServers] = useState([]);
  const [loading, setLoading] = useState(true);

  const fetchServers = async () => {
    const { data, error } = await supabase
      .from('iblups_servers_status')
      .select('*')
      .order('is_active', { ascending: false });

    if (error) {
      console.error('Error:', error);
    } else {
      setServers(data || []);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchServers();

    // Actualizar cada 5 segundos
    const interval = setInterval(fetchServers, 5000);
    return () => clearInterval(interval);
  }, []);

  if (loading) return Cargando...;

  return (




            Servidor
            Ubicación
            Estado
            CPU
            Memoria
            Streams
            Conexiones



          {servers.map((server) => (

              {server.server_name}
              {server.location}


                  {server.is_active ? '🟢 Online' : '🔴 Offline'}


              {server.current_cpu.toFixed(1)}%
              {server.current_memory} MB
              {server.current_streams}
              {server.current_connections}

          ))}



  );
}
```

---

### Componente: Gráfico de CPU (últimas 6 horas)

```typescript
// components/CPUChart.tsx
'use client';

import { useEffect, useState } from 'react';
import { supabase } from '@/lib/supabase';
import { Line } from 'react-chartjs-2';

export default function CPUChart() {
  const [chartData, setChartData] = useState(null);

  useEffect(() => {
    const fetchCPUData = async () => {
      const sixHoursAgo = new Date(Date.now() - 6 * 60 * 60 * 1000);

      const { data, error } = await supabase
        .from('iblups_server_metrics')
        .select('timestamp, server_id, cpu_percent')
        .gte('timestamp', sixHoursAgo.toISOString())
        .order('timestamp', { ascending: true });

      if (error) {
        console.error('Error:', error);
        return;
      }

      // Agrupar por servidor
      const serverIds = [...new Set(data.map(d => d.server_id))];
      const labels = [...new Set(data.map(d =>
        new Date(d.timestamp).toLocaleTimeString()
      ))];

      const datasets = serverIds.map(serverId => ({
        label: serverId,
        data: data
          .filter(d => d.server_id === serverId)
          .map(d => d.cpu_percent),
        borderColor: getRandomColor(),
        tension: 0.4,
      }));

      setChartData({ labels, datasets });
    };

    fetchCPUData();
    const interval = setInterval(fetchCPUData, 30000); // Actualizar cada 30s
    return () => clearInterval(interval);
  }, []);

  if (!chartData) return Cargando gráfico...;

  return (



  );
}

function getRandomColor() {
  const colors = [
    'rgb(59, 130, 246)',   // blue
    'rgb(34, 197, 94)',    // green
    'rgb(239, 68, 68)',    // red
    'rgb(168, 85, 247)',   // purple
    'rgb(251, 146, 60)',   // orange
  ];
  return colors[Math.floor(Math.random() * colors.length)];
}
```

---

### Componente: KPIs Dashboard

```typescript
// components/KPIDashboard.tsx
'use client';

import { useEffect, useState } from 'react';
import { supabase } from '@/lib/supabase';

interface KPIs {
  totalServers: number;
  activeServers: number;
  totalStreams: number;
  totalViewers: number;
  avgCPU: number;
  totalBandwidth: number;
}

export default function KPIDashboard() {
  const [kpis, setKpis] = useState(null);

  useEffect(() => {
    const fetchKPIs = async () => {
      // Query para obtener últimas métricas de cada servidor
      const { data: metrics } = await supabase
        .from('iblups_server_metrics')
        .select('*')
        .gte('timestamp', new Date(Date.now() - 60000).toISOString());

      if (!metrics) return;

      // Obtener última métrica por servidor
      const latestByServer = metrics.reduce((acc, m) => {
        if (!acc[m.server_id] || new Date(m.timestamp) > new Date(acc[m.server_id].timestamp)) {
          acc[m.server_id] = m;
        }
        return acc;
      }, {} as Record);

      const latest = Object.values(latestByServer);

      const totalServers = latest.length;
      const activeServers = latest.filter((m: any) =>
        (Date.now() - new Date(m.timestamp).getTime()) < 60000
      ).length;
      const totalStreams = latest.reduce((sum: number, m: any) =>
        sum + m.total_streams, 0
      );
      const totalViewers = latest.reduce((sum: number, m: any) =>
        sum + m.players, 0
      );
      const avgCPU = latest.reduce((sum: number, m: any) =>
        sum + m.cpu_percent, 0
      ) / latest.length;

      setKpis({
        totalServers,
        activeServers,
        totalStreams,
        totalViewers,
        avgCPU,
        totalBandwidth: 0, // Calcular según necesites
      });
    };

    fetchKPIs();
    const interval = setInterval(fetchKPIs, 5000);
    return () => clearInterval(interval);
  }, []);

  if (!kpis) return Cargando KPIs...;

  return (






  );
}

function KPICard({ title, value, color }: any) {
  const colors = {
    blue: 'bg-blue-50 text-blue-700 border-blue-200',
    green: 'bg-green-50 text-green-700 border-green-200',
    purple: 'bg-purple-50 text-purple-700 border-purple-200',
    orange: 'bg-orange-50 text-orange-700 border-orange-200',
  };

  return (

      {title}
      {value}

  );
}
```

---

## 🔄 Real-time con Supabase Subscriptions

```typescript
// hooks/useRealtimeMetrics.ts
import { useEffect, useState } from 'react';
import { supabase } from '@/lib/supabase';

export function useRealtimeMetrics() {
  const [latestMetric, setLatestMetric] = useState(null);

  useEffect(() => {
    // Suscribirse a nuevas inserciones en server_metrics
    const subscription = supabase
      .channel('server_metrics_channel')
      .on(
        'postgres_changes',
        {
          event: 'INSERT',
          schema: 'public',
          table: 'iblups_server_metrics'
        },
        (payload) => {
          console.log('Nueva métrica:', payload.new);
          setLatestMetric(payload.new);
        }
      )
      .subscribe();

    return () => {
      subscription.unsubscribe();
    };
  }, []);

  return latestMetric;
}

// Uso en componente
function MyComponent() {
  const latestMetric = useRealtimeMetrics();

  return (

      {latestMetric && (

          Servidor: {latestMetric.server_id}
          CPU: {latestMetric.cpu_percent}%

      )}

  );
}
```

---

## 📈 Casos de Uso Comunes

### 1. Mapa de Calor de Uso por Hora del Día

```sql
SELECT
    EXTRACT(HOUR FROM timestamp) as hour,
    EXTRACT(DOW FROM timestamp) as day_of_week,
    ROUND(AVG(total_connections)::numeric, 0) as avg_connections
FROM iblups_server_metrics
WHERE timestamp >= NOW() - INTERVAL '30 days'
GROUP BY EXTRACT(HOUR FROM timestamp), EXTRACT(DOW FROM timestamp)
ORDER BY day_of_week, hour;
```

---

### 2. Detección de Anomalías (CPU inusualmente alta)

```sql
WITH stats AS (
    SELECT
        server_id,
        AVG(cpu_percent) as avg_cpu,
        STDDEV(cpu_percent) as stddev_cpu
    FROM iblups_server_metrics
    WHERE timestamp >= NOW() - INTERVAL '7 days'
    GROUP BY server_id
)
SELECT
    m.timestamp,
    m.server_id,
    m.cpu_percent,
    s.avg_cpu,
    (m.cpu_percent - s.avg_cpu) / NULLIF(s.stddev_cpu, 0) as z_score
FROM iblups_server_metrics m
JOIN stats s ON m.server_id = s.server_id
WHERE m.timestamp >= NOW() - INTERVAL '1 hour'
    AND ABS((m.cpu_percent - s.avg_cpu) / NULLIF(s.stddev_cpu, 0)) > 2
ORDER BY ABS((m.cpu_percent - s.avg_cpu) / NULLIF(s.stddev_cpu, 0)) DESC;
```

---

### 3. Predicción de Capacidad

```sql
-- Crecimiento de viewers en los últimos 30 días
WITH daily_stats AS (
    SELECT
        DATE(timestamp) as date,
        MAX(players) as max_viewers
    FROM iblups_server_metrics
    WHERE timestamp >= NOW() - INTERVAL '30 days'
    GROUP BY DATE(timestamp)
)
SELECT
    date,
    max_viewers,
    AVG(max_viewers) OVER (ORDER BY date ROWS BETWEEN 6 PRECEDING AND CURRENT ROW) as moving_avg_7d
FROM daily_stats
ORDER BY date DESC;
```

---

### 4. Costo Estimado por Bandwidth

```sql
SELECT
    server_id,
    DATE(timestamp) as date,
    SUM(send_kbps) * 30 / 1024 / 8 as estimated_gb_sent_per_30s,
    -- Si el costo es $0.05/GB
    (SUM(send_kbps) * 30 / 1024 / 8) * 0.05 as estimated_cost_usd
FROM iblups_stream_metrics
WHERE timestamp >= NOW() - INTERVAL '1 day'
GROUP BY server_id, DATE(timestamp)
ORDER BY estimated_cost_usd DESC;
```

---

## 🔧 Mantenimiento y Limpieza

### Ejecutar limpieza automática (cron job)

```sql
-- Ejecutar manualmente
SELECT cleanup_old_iblups_metrics();

-- O crear un cron job en Supabase (pg_cron extension)
SELECT cron.schedule(
    'cleanup-old-metrics',
    '0 2 * * *',  -- Cada día a las 2 AM
    $$SELECT cleanup_old_iblups_metrics()$$
);
```

### Marcar servidores inactivos

```sql
SELECT mark_inactive_servers();
```

---

## 🎯 Próximos Pasos

1. **Implementar alertas por email/Slack** cuando CPU > 90%
2. **Dashboard de comparación** entre servidores
3. **Predicción de carga** con machine learning
4. **Auto-scaling** basado en métricas
5. **Geolocalización** de viewers por IP

---

## 📞 Soporte

Para dudas sobre la implementación del dashboard:

- Backend: Revisa `/api/v1/*` endpoints
- Base de datos: Consulta las vistas preconstruidas
- Real-time: Usa Supabase subscriptions

**Última actualización:** 2026-02-06
