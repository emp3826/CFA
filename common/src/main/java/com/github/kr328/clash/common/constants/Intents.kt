package com.github.kr328.clash.common.constants

object Intents {
    // Public
    val ACTION_PROVIDE_URL = "action.PROVIDE_URL"

    const val EXTRA_NAME = "name"

    // Self
    val ACTION_SERVICE_RECREATED = "intent.action.CLASH_RECREATED"
    val ACTION_CLASH_STARTED = "intent.action.CLASH_STARTED"
    val ACTION_CLASH_STOPPED = "intent.action.CLASH_STOPPED"
    val ACTION_CLASH_REQUEST_STOP = "intent.action.CLASH_REQUEST_STOP"
    val ACTION_PROFILE_CHANGED = "intent.action.PROFILE_CHANGED"
    val ACTION_PROFILE_REQUEST_UPDATE = "intent.action.REQUEST_UPDATE"
    val ACTION_PROFILE_SCHEDULE_UPDATES = "intent.action.SCHEDULE_UPDATES"
    val ACTION_PROFILE_LOADED = "intent.action.PROFILE_LOADED"
    val ACTION_OVERRIDE_CHANGED = "intent.action.OVERRIDE_CHANGED"

    const val EXTRA_STOP_REASON = "stop_reason"
    const val EXTRA_UUID = "uuid"
}
