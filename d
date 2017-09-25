#!/bin/bash

cd "$(dirname $0)"

task=$1 # More descriptive name
arg=$2
args=${*:2}

appname=bigbrother

case $task in
    build)
        docker build -t $appname .
        ;;
    run)
        if ! docker inspect $appname-postgres > /dev/null 2> /dev/null; then
            docker run -d \
                --name $appname-postgres \
                postgres:9.6
        fi

        docker start $appname-postgres
        sleep 2

        BB_POSTGRES_HOST=$(docker inspect --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' bigbrother-postgres) \
        BB_POSTGRES_PORT=5432 \
        BB_POSTGRES_DATABASE=postgres \
        BB_POSTGRES_USER=postgres \
        BB_POSTGRES_PASSWORD=postgres \
        BB_STORAGE_PATH=storage \
        BigBrother
        ;;
    start)
        if ! docker inspect $appname-postgres > /dev/null 2> /dev/null; then
            docker run -d \
                --name $appname-postgres \
                postgres:9.6
        fi

        docker start $appname-postgres
        sleep 2

        docker rm -f $appname || true
        docker run \
            -i -t --rm \
            --name $appname \
            --link $appname-postgres:postgres \
            -p 0.0.0.0:3000:3000 \
            -v $(pwd)/storage:/var/bigbrother:rw \
            -e BB_POSTGRES_HOST=postgres \
            -e BB_POSTGRES_PORT=5432 \
            -e BB_POSTGRES_DATABASE=postgres \
            -e BB_POSTGRES_USER=postgres \
            -e BB_POSTGRES_PASSWORD=postgres \
            -e BB_STORAGE_PATH=/var/bigbrother \
            $appname
        ;;
    stop)
        docker stop $appname
        docker stop $appname-postgres
        ;;
    shell)
        docker exec -i -t $appname bash
        ;;
    dbshell)
        docker exec -i -t $appname-postgres psql -U postgres
        ;;
    *)
        echo 'Usage: ./d action [params]. For a list of actions, read the d file'
        ;;
esac
