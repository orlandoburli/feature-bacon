FROM python:3.12-slim
RUN useradd -r -s /usr/sbin/nologin appuser
WORKDIR /app
COPY sdks/python/ /app/sdks/python/
COPY samples/07-sdk-python/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt && pip install --no-cache-dir /app/sdks/python/
COPY samples/07-sdk-python/app.py ./
USER appuser
EXPOSE 3000
CMD ["python", "app.py"]
