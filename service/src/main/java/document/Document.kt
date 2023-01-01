package com.github.kr328.clash.service.document

interface Document {
    val id: String
    val name: String
    val mimeType: String
    val flags: Set<Flag>
}
