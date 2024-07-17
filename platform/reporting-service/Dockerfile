# Use an official Python runtime as a parent image
FROM python:3.12-slim

# Set the working directory to /code
WORKDIR /code

# Copy the current directory contents into the container at /code
COPY requirements.txt /code/

# Install any needed packages specified in requirements.txt
# Adding `--default-timeout=100` might help with some network issues during build
RUN pip install --no-cache-dir --upgrade pip \
    && pip install --no-cache-dir --upgrade -r requirements.txt --default-timeout=100

# Copy the rest of your application's code
COPY ./app /code/app

# Make port 8000 available to the world outside this container
EXPOSE 8000

# Define environment variable
ENV PYTHONUNBUFFERED 1

# Run uvicorn when the container launches
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
