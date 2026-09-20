# The prod edge: pure router, no content of its own. Routes /api to the
# CarePath api (with the mock-his console carve-outs), /console to mock-his,
# and everything else to the static web tier — see proxy.conf.
FROM nginx:1.27-alpine
COPY infra/docker/proxy.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
