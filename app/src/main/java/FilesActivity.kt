@file:Suppress("BlockingMethodInNonBlockingContext")

package com.github.kr328.clash

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat
import com.github.kr328.clash.common.util.grantPermissions
import com.github.kr328.clash.common.util.ticker
import com.github.kr328.clash.common.util.uuid
import com.github.kr328.clash.design.FilesDesign
import com.github.kr328.clash.remote.FilesClient
import com.github.kr328.clash.service.model.Profile
import com.github.kr328.clash.util.fileName
import com.github.kr328.clash.util.withProfile
import kotlinx.coroutines.isActive
import kotlinx.coroutines.selects.select
import java.util.Stack

class FilesActivity : BaseActivity<FilesDesign>() {
    override suspend fun main() {
        val uuid = intent.uuid ?: return finish()
        val profile = withProfile { queryByUUID(uuid) } ?: return finish()
        val root = uuid.toString()

        val design = FilesDesign(this)
        val client = FilesClient(this)
        val stack = Stack<String>()

        design.configurationEditable = profile.type != Profile.Type.Url
        design.fetch(client, stack, root)

        setContentDesign(design)

        while (isActive) {
            select<Unit> {
                events.onReceive {
                    when (it) {
                        Event.ActivityStart, Event.ActivityStop -> {
                            design.fetch(client, stack, root)
                        }
                        else -> Unit
                    }
                }
                design.requests.onReceive {
                    try {
                        when (it) {
                            FilesDesign.Request.PopStack -> {
                                if (stack.empty()) {
                                    finish()
                                } else {
                                    stack.pop()
                                }
                            }
                            is FilesDesign.Request.OpenDirectory -> {
                                stack.push(it.file.id)
                            }
                            is FilesDesign.Request.OpenFile -> {
                                startActivityForResult(
                                    ActivityResultContracts.StartActivityForResult(),
                                    Intent(Intent.ACTION_VIEW).setDataAndType(
                                        client.buildDocumentUri(it.file.id),
                                        "text/plain"
                                    ).grantPermissions()
                                )
                            }
                            is FilesDesign.Request.DeleteFile -> {
                                client.deleteDocument(it.file.id)
                            }
                            is FilesDesign.Request.RenameFile -> {
                                val newName = design.requestFileName(it.file.name)

                                client.renameDocument(it.file.id, newName)
                            }
                            is FilesDesign.Request.ImportFile -> {
                                if (Build.VERSION.SDK_INT >= 23) {
                                    val hasPermission = ContextCompat.checkSelfPermission(
                                        this@FilesActivity,
                                        Manifest.permission.READ_EXTERNAL_STORAGE
                                    ) == PackageManager.PERMISSION_GRANTED

                                    if (!hasPermission) {
                                        val granted = startActivityForResult(
                                            ActivityResultContracts.RequestPermission(),
                                            Manifest.permission.READ_EXTERNAL_STORAGE,
                                        )

                                        if (!granted) {
                                            return@onReceive
                                        }
                                    }
                                }

                                val uri: Uri? = startActivityForResult(
                                    ActivityResultContracts.GetContent(),
                                    "*/*"
                                )

                                if (uri != null) {
                                    if (it.file == null) {
                                        val name = design.requestFileName(uri.fileName ?: "File")

                                        client.importDocument(stack.last(), uri, name)
                                    }
                                }
                            }
                        }
                    } catch (e: Exception) {
                    }

                    design.fetch(client, stack, root)
                }
            }
        }
    }

    override fun onBackPressed() {
        design?.requests?.trySend(FilesDesign.Request.PopStack)
    }

    private suspend fun FilesDesign.fetch(client: FilesClient, stack: Stack<String>, root: String) {
        val documentId = stack.lastOrNull() ?: root
        val files = if (stack.empty()) {
            val list = client.list(documentId)
            val config = list.firstOrNull { it.id.endsWith("config.yaml") }

            if (config == null || config.size > 0) list else listOf(config)
        } else {
            client.list(documentId)
        }

        swapFiles(files, stack.empty())
    }
}
