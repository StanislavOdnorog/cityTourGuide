FROM postgres:16-alpine

RUN apk add --no-cache \
    aws-cli \
    bash \
    curl

COPY backup.sh /usr/local/bin/backup.sh
COPY restore.sh /usr/local/bin/restore.sh

RUN chmod +x /usr/local/bin/backup.sh /usr/local/bin/restore.sh

# Usage:
#   docker run csg-backup                                    # Run backup
#   docker run csg-backup /usr/local/bin/restore.sh --list   # List backups
#   docker run csg-backup /usr/local/bin/restore.sh          # Restore latest
CMD ["/usr/local/bin/backup.sh"]
