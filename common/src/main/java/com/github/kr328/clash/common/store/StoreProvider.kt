package com.github.kr328.clash.common.store

interface StoreProvider {
    fun getInt(key: String, defaultValue: Int): Int
    fun setInt(key: String, value: Int)

    fun getString(key: String, defaultValue: String): String
    fun setString(key: String, value: String)

    fun getBoolean(key: String, defaultValue: Boolean): Boolean
    fun setBoolean(key: String, value: Boolean)
}
