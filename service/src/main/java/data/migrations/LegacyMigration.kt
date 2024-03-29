@file:Suppress("BlockingMethodInNonBlockingContext")

package com.github.kr328.clash.service.data.migrations

import android.content.Context
import android.database.sqlite.SQLiteDatabase
import androidx.core.text.isDigitsOnly
import com.github.kr328.clash.service.data.Pending
import com.github.kr328.clash.service.data.PendingDao
import com.github.kr328.clash.service.model.Profile
import com.github.kr328.clash.service.util.generateProfileUUID
import com.github.kr328.clash.service.util.pendingDir
import com.github.kr328.clash.service.util.sendProfileChanged
import java.io.File

internal suspend fun migrationFromLegacy(context: Context) {
    val file = context.getDatabasePath("clash-config")

    if (!file.exists()) {
        return
    }

    try {
        SQLiteDatabase.openDatabase(file.absolutePath, null, SQLiteDatabase.OPEN_READONLY)
            .use { db ->
                val v = db.version

                migrationFromLegacy234(context, db)
            }
    } catch (e: Exception) {
        return
    }

    context.deleteDatabase("clash-config")
}

private suspend fun migrationFromLegacy234(context: Context, legacy: SQLiteDatabase) {
    legacy.query(
        "profiles",
        arrayOf("id", "name", "type", "uri"),
        null,
        null,
        null,
        null,
        "id"
    ).use { cursor ->
        val id = cursor.getColumnIndex("id")
        val name = cursor.getColumnIndex("name")
        val type = cursor.getColumnIndex("type")
        val uri = cursor.getColumnIndex("uri")
        if (!cursor.moveToFirst())
            return

        do {
            val newType = when (cursor.getInt(type)) {
                1 -> { // TYPE_FILE
                    Profile.Type.File
                }
                2 -> { // TYPE_URL
                    Profile.Type.Url
                }
                3 -> { // TYPE_EXTERNAL
                    Profile.Type.External
                }
                else -> { // unknown
                    continue
                }
            }

            val idValue = cursor.getInt(id)

            val pending = Pending(
                uuid = generateProfileUUID(),
                name = cursor.getString(name),
                type = newType,
                source = if (newType != Profile.Type.File) cursor.getString(uri) else "",
            )

            val base = context.pendingDir.resolve(pending.uuid.toString())

            base.apply {
                mkdirs()

                resolve("config.yaml").createNewFile()
                resolve("providers").mkdir()
            }

            if (newType == Profile.Type.File) {
                val legacyFile = context.filesDir.resolve("profiles/$idValue.yaml")

                if (legacyFile.isFile) {
                    legacyFile.copyTo(base.resolve("config.yaml"), overwrite = true)
                }
            }

            PendingDao().insert(pending)

            context.sendProfileChanged(pending.uuid)

        } while (cursor.moveToNext())
    }

    context.filesDir.resolve("profiles").deleteRecursively()
    context.filesDir.resolve("clash").listFiles()?.forEach {
        if (it.name.isDigitsOnly()) {
            it.deleteRecursively()
        }
    }
}
