#include "bridge.h"

void (*mark_socket_func)(void *tun_interface, int fd);

int (*query_socket_uid_func)(void *tun_interface, int protocol, const char *source, const char *target);

void (*complete_func)(void *completable, const char *exception);

void (*fetch_report_func)(void *fetch_callback, const char *status_json);

void (*fetch_complete_func)(void *fetch_callback, const char *error);

int (*open_content_func)(const char *url, char *error, int error_length);

void (*release_object_func)(void *obj);

void mark_socket(void *interface, int fd) {
    mark_socket_func(interface, fd);
}

int query_socket_uid(void *interface, int protocol, char *source, char *target) {
    int result = query_socket_uid_func(interface, protocol, source, target);

    free(source);
    free(target);

    return result;
}

void complete(void *obj, char *error) {
    complete_func(obj, error);

    free(error);
}

void fetch_complete(void *fetch_callback, char *exception) {
    fetch_complete_func(fetch_callback, exception);

    free(exception);
}

void fetch_report(void *fetch_callback, char *json_status) {
    fetch_report_func(fetch_callback, json_status);

    free(json_status);
}

int open_content(char *url, char *error, int error_length) {
    int result = open_content_func(url, error, error_length);

    free(url);

    return result;
}

void release_object(void *obj) {
    release_object_func(obj);
}
