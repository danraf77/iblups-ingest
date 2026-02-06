# **🚀 SRS Streaming Cluster \+ Go Backend**

### **_Optimizado para Infraestructura OVH (16 vCores)_**

Esta solución integral configura un nodo de transmisión de video de **alta disponibilidad** utilizando **SRS (Simple Realtime Server) v6**. El sistema está diseñado específicamente para aprovechar servidores con múltiples núcleos, garantizando baja latencia, seguridad de las claves de transmisión y gestión automática de estados en **Supabase**.

---

## **✨ Características Técnicas**

- **Ingesta Masiva:** Optimización de descriptores de archivos y stack de red para superar el límite estándar de 60 conexiones.
- **Seguridad "Zero-Exposure":** El `stream_id` (clave de OBS) nunca se expone al público.
- **Thumbnails Persistentes:** Generación de miniaturas vía FFmpeg con hash MD5 basado en el UUID del canal.
- **Backend Asíncrono en Go:** Las actualizaciones en la base de datos no bloquean el flujo de video (latencia cero).
- **Modo Relay Pure:** Configurado para reenvío directo (_pass-through_) al servidor HLS externo `37.59.97.144`.

---

## **📂 Estructura del Proyecto**

Plaintext  
.  
├── .env \# Variables de entorno  
├── .gitignore \# Archivos excluidos  
├── Dockerfile \# Receta Go \+ FFmpeg  
├── README.md \# Este archivo  
├── docker-compose.yml \# Orquestación Docker  
├── main.go \# Backend en Go  
├── srs.conf \# Configuración SRS  
└── thumbnails/ \# Carpeta de miniaturas

---

## **🛠️ Instalación y Configuración del VPS**

### **1\. Preparación del Sistema Operativo**

Ejecuta estos comandos en tu VPS para optimizar el Kernel:

Bash  
\# Actualizar e instalar dependencias  
sudo apt update && sudo apt upgrade \-y  
sudo apt install docker.io docker-compose git \-y

\# Optimizar el Kernel (Ulimits y Red)  
sudo bash \-c 'cat \<\<EOF \>\> /etc/sysctl.conf  
fs.file-max=200000  
net.core.somaxconn=8192  
net.ipv4.tcp_max_syn_backlog=8192  
net.ipv4.ip_local_port_range=1024 65535  
EOF'  
sudo sysctl \-p

### **2\. Despliegue de Servicios**

Bash  
\# Crear directorio y entrar  
mkdir \-p /opt/srs-streaming/thumbnails && cd /opt/srs-streaming  
chmod 777 thumbnails

\# Levantar contenedores  
docker-compose up \-d \--build

---

## **🖼️ Distribución de Miniaturas (Nginx)**

Configura Nginx para servir las imágenes generadas por FFmpeg:

Nginx  
server {  
 listen 80;  
 server_name tu-dominio-o-ip.com;

    location /thumbs/ {
        alias /opt/srs-streaming/thumbnails/;
        add\_header Cache-Control "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0";
        add\_header Access-Control-Allow-Origin \*;
    }

}

---

## **🔒 Flujo de Seguridad**

1. **Streamer:** Envía señal a `rtmp://TU_IP/live/clave_secreta`.
2. **Backend Go:** Valida la clave y busca el UUID en Supabase.
3. **FFmpeg:** Captura un frame cada 10s y lo guarda como `hash_md5(uuid).jpg`.
4. **Frontend:** Muestra la imagen pública sin revelar la clave de transmisión.
