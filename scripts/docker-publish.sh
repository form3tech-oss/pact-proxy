#!/bin/bash

function publish() {
    echo "Publishing to $1"
    if ! aws ecr describe-repositories --repository-names tech.form3/$NAME --region $1 > /dev/null 2>&1 ; then
        aws ecr create-repository --repository-name tech.form3/$NAME --region $1
    fi

    eval $(aws ecr get-login --region $1 --no-include-email)

    docker tag tech.form3/$NAME:$TAG 288840537196.dkr.ecr.$1.amazonaws.com/tech.form3/$NAME:$TAG
    docker push 288840537196.dkr.ecr.$1.amazonaws.com/tech.form3/$NAME:$TAG
}

NAME=$(basename $1)

publish eu-west-1
publish eu-west-2
