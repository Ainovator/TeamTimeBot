FROM python:3.12-slim

WORKDIR /app
COPY server.py /app/server.py

ENV PYTHONUNBUFFERED=1
EXPOSE 9000

CMD ["python", "/app/server.py"]

