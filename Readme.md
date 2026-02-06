🚀 SRS Streaming Cluster + Go Backend (Optimizado para OVH)
Esta solución integral configura un nodo de transmisión de video de alta disponibilidad utilizando SRS (Simple Realtime Server) v6. El sistema está diseñado específicamente para aprovechar servidores con múltiples núcleos (como el VPS de 16 vCores de OVH), garantizando baja latencia, seguridad de las claves de transmisión y gestión automática de estados en Supabase.

✨ Características Técnicas
Ingesta Masiva: Optimización de descriptores de archivos y stack de red para superar el límite estándar de 60 conexiones.

Seguridad "Zero-Exposure": El stream_id (clave de OBS) nunca se expone al público.

Thumbnails Persistentes: Generación de miniaturas vía FFmpeg con hash MD5 basado en el UUID del canal.

Backend Asíncrono en Go: Las actualizaciones en la base de datos no bloquean el flujo de video (latencia cero).

Modo Relay Pure: Configurado para reenvío directo (pass-through) al servidor HLS externo 37.59.97.144 sin carga innecesaria de transcoding.

📂 Estructura del Proyecto
Plaintext
.
├── .env # Variables de entorno (Credenciales sensibles)
├── .gitignore # Archivos excluidos del control de versiones
├── Dockerfile # Receta para la imagen Go + FFmpeg
├── README.md # Documentación completa (este archivo)
├── docker-compose.yml # Orquestación de contenedores Docker
├── main.go # Backend de lógica y controladores en Go
├── srs.conf # Configuración del servidor de medios SRS
└── thumbnails/ # Carpeta persistente para las miniaturas (Auto-generada)
🛠️ Instalación y Configuración del VPS

1. Preparación del Sistema Operativo (Host)
   Es fundamental preparar el kernel de Linux para manejar el tráfico de video masivo. Ejecuta estos comandos en tu VPS:

Bash

# Actualizar el sistema e instalar Docker

sudo apt update && sudo apt upgrade -y
sudo apt install docker.io docker-compose nginx -y

# Optimizar el Kernel (Ulimits y Red)

sudo bash -c 'cat <<EOF >> /etc/sysctl.conf
fs.file-max=200000
net.core.somaxconn=8192
net.ipv4.tcp_max_syn_backlog=8192
net.ipv4.ip_local_port_range=1024 65535
EOF'
sudo sysctl -p 2. Configuración de Directorios
Bash
mkdir -p /opt/srs-streaming/thumbnails && cd /opt/srs-streaming
chmod 777 thumbnails 3. Configuración del Entorno (.env)
Crea un archivo llamado .env y rellena con tus datos reales:

Fragmento de código
SUPABASE_URL=https://tu-id.supabase.co
SUPABASE_KEY=tu-anon-key-de-supabase
TARGET_FORWARD_URL=rtmp://37.59.97.144:1935/live 4. Despliegue de Servicios
Ejecuta el comando para construir e iniciar los contenedores en segundo plano:

Bash
docker-compose up -d --build
🖼️ Distribución de Miniaturas (Nginx)
Para que los thumbnails sean accesibles de forma eficiente, configuramos Nginx en el host como servidor de archivos estáticos:

Crea la configuración: sudo nano /etc/nginx/sites-available/streaming

Pega el siguiente contenido:

Nginx
server {
listen 80;
server_name tu-dominio-o-ip.com;

    location /thumbs/ {
        alias /opt/srs-streaming/thumbnails/;
        # Evitar caché para que se vea la actualización cada 10 seg
        add_header Cache-Control "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0";
        add_header Access-Control-Allow-Origin *;
        expires off;
        etag off;
    }

}
Activa la configuración y reinicia:

Bash
sudo ln -s /etc/nginx/sites-available/streaming /etc/nginx/sites-enabled/
sudo systemctl restart nginx
🔒 Flujo de Operación y Seguridad
Conexión: El streamer publica en rtmp://TU_IP/live/clave_secreta.

Autorización: SRS notifica al backend en Go. Go responde con 0 inmediatamente para que el stream inicie sin lag.

Identificación: Go busca en la tabla channels_channel el UUID correspondiente al stream_id recibido.

Thumbnail:

Se genera un hash MD5 del UUID (ej: a1b2c3...jpg).

Se inicia un proceso FFmpeg que captura un frame cada 10 segundos.

La imagen se sobrescribe en la carpeta /thumbnails de forma persistente.

Base de Datos: Se actualiza is_on_live = true y cover = a1b2c3...jpg.

Frontend: El cliente solicita http://TU_IP/thumbs/a1b2c3...jpg, protegiendo la clave_secreta.

📊 Monitoreo y Mantenimiento
Logs del Backend: docker logs -f backend-go

Estadísticas de SRS: Accede a http://TU_IP:1985/api/v1/summaries para ver conexiones activas.

Limpieza: La carpeta thumbnails/ se mantiene limpia automáticamente ya que el sistema sobrescribe el archivo existente para cada canal.

📝 Notas sobre la Base de Datos
El sistema interactúa con la tabla channels_channel de Supabase, actualizando específicamente:

is_on_live: (boolean) Estado actual del stream.

last_status: (string) "online" u "offline".

cover: (string) Nombre del archivo hash generado.

modified: (timestamp) Fecha de la última actualización.
