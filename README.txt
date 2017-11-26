This tool will stop/start a go service on a given server.

If local resources (such as ports) are used it will present them to the
application in form of variables.

for example, a command line might be defined as "-port=${PORT1}" where ${PORT1} is a variable determined in runtime by the autodeployer.

it is intented to download stuff from the build-repository
