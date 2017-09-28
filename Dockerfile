FROM python:2.7-onbuild

# Bootstrap application folders
RUN mkdir -p /app /app/conf /app/data

# Copy requirements.txt && run pip install
COPY requirements.txt /app/
RUN pip install --no-cache-dir -r /app/requirements.txt

# Copy application files
COPY conf/bot.yaml /app/conf/
COPY botoftheday /app/
COPY generate.py /app/

# Set the default directory where CMD will execute
WORKDIR /app

# Run command
CMD ["/bin/bash"]
