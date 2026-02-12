package com.trah.abnotify.ui

import android.os.Bundle
import android.view.View
import android.widget.AdapterView
import android.widget.ArrayAdapter
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import com.trah.abnotify.AbnotifyApp
import com.trah.abnotify.databinding.ActivityTutorialBinding
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.net.HttpURLConnection
import java.net.URL

class TutorialActivity : AppCompatActivity() {

    private lateinit var binding: ActivityTutorialBinding
    private var isGetRequest = false

    private val examples = listOf(
        "选择示例模板...",
        "GET 请求示例",
        "POST 简单消�?,
        "POST 带标题消�?,
        "POST 多行消息"
    )

    // Map to track which examples are GET requests
    private val getRequestExamples = setOf("GET 请求示例")

    private fun getExampleMessage(exampleName: String): String {
        val pushUrl = getPushUrl()
        return when (exampleName) {
            "GET 请求示例" -> "${pushUrl}/测试标题/这是一条测试消�?
            "POST 简单消�? -> """{
  "body": "这是一条简单的推送消�?
}"""
            "POST 带标题消�? -> """{
  "title": "服务器告�?,
  "body": "CPU 使用率已�?95%"
}"""
            "POST 多行消息" -> """{
  "title": "任务完成",
  "body": "备份任务已完成\n耗时: 5分钟\n大小: 1.2GB"
}"""
            else -> ""
        }
    }



    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityTutorialBinding.inflate(layoutInflater)
        setContentView(binding.root)

        setupUI()
        setupInstructions()
    }


    private fun setupUI() {
        // Back button
        binding.btnBack.setOnClickListener {
            finish()
        }

        // Spinner setup
        val adapter = ArrayAdapter(this, android.R.layout.simple_spinner_item, examples)
        adapter.setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item)
        binding.spinnerExamples.adapter = adapter

        binding.spinnerExamples.onItemSelectedListener = object : AdapterView.OnItemSelectedListener {
            override fun onItemSelected(parent: AdapterView<*>?, view: View?, position: Int, id: Long) {
                if (position > 0) {
                    val selectedExample = examples[position]
                    isGetRequest = getRequestExamples.contains(selectedExample)
                    val message = getExampleMessage(selectedExample)
                    binding.etMessage.setText(message)

                    // Update button text based on request type
                    binding.btnSendTest.text = if (isGetRequest) "发�?GET 请求" else "发�?POST 请求"
                }
            }

            override fun onNothingSelected(parent: AdapterView<*>?) {}
        }

        // Send test button
        binding.btnSendTest.setOnClickListener {
            sendTestPush()
        }
    }


    private fun getPushUrl(): String {
        val keyManager = AbnotifyApp.getInstance().keyManager
        val serverUrl = keyManager.serverUrl
        val deviceKey = keyManager.getDeviceKey() ?: "YOUR_DEVICE_KEY"
        return "$serverUrl/push/$deviceKey"
    }

    private fun setupInstructions() {
        val pushUrl = getPushUrl()
        
        val markdown = """
## 什么是 abnotify

abnotify 是一款基�?**WebSocket** 的安卓推送工具。通过简单的 HTTP 请求，即可向手机发送实时通知�?

---

## 快速开�?

1. 复制首页�?**Push URL**
2. 向该地址发�?HTTP 请求
3. 手机即时收到推送通知

---

## 请求参数

- `title` - 通知标题（可选）
- `body` - 通知内容�?*必填**�?

---

## 请求方式

**GET 请求（不加密，仅测试用）**
```
${pushUrl}/标题/内容
```

**POST 请求（RSA 加密�?*
```
POST ${pushUrl}
Content-Type: application/json

{"title": "标题", "body": "内容"}
```

---

## 安全说明

- **POST 请求**：RSA 端到端加�?
- **GET 请求**：明文传输，仅用于测�?
- Push Key 是唯一凭证，请妥善保管
- 重置密钥后，旧链接立即失�?

---

## 应用场景

- 服务器监控告�?
- CI/CD 构建通知
- 自动化脚本通知
- IoT 设备状态推�?
- 定时任务提醒
        """.trimIndent()

        
        val markwon = io.noties.markwon.Markwon.create(this)
        markwon.setMarkdown(binding.tvInstructions, markdown)
    }


    private fun sendTestPush() {
        val message = binding.etMessage.text.toString().trim()
        if (message.isEmpty()) {
            Toast.makeText(this, "请输入消息内�?, Toast.LENGTH_SHORT).show()
            return
        }

        binding.btnSendTest.isEnabled = false
        binding.btnSendTest.text = "发送中..."

        CoroutineScope(Dispatchers.IO).launch {
            try {
                val responseCode: Int
                
                if (isGetRequest) {
                    // GET request - message is the full URL
                    val url = URL(message)
                    val connection = url.openConnection() as HttpURLConnection
                    connection.requestMethod = "GET"
                    connection.connectTimeout = 10000
                    connection.readTimeout = 10000
                    responseCode = connection.responseCode
                    connection.disconnect()
                } else {
                    // POST request - message is JSON body
                    val url = URL(getPushUrl())
                    val connection = url.openConnection() as HttpURLConnection
                    connection.requestMethod = "POST"
                    connection.setRequestProperty("Content-Type", "application/json; charset=UTF-8")
                    connection.doOutput = true
                    connection.connectTimeout = 10000
                    connection.readTimeout = 10000

                    connection.outputStream.use { os ->
                        os.write(message.toByteArray(Charsets.UTF_8))
                    }
                    responseCode = connection.responseCode
                    connection.disconnect()
                }

                withContext(Dispatchers.Main) {
                    binding.btnSendTest.isEnabled = true
                    binding.btnSendTest.text = if (isGetRequest) "发�?GET 请求" else "发�?POST 请求"

                    if (responseCode in 200..299) {
                        Toast.makeText(this@TutorialActivity, "推送成功！请查看通知", Toast.LENGTH_SHORT).show()
                    } else {
                        Toast.makeText(this@TutorialActivity, "推送失�? HTTP $responseCode", Toast.LENGTH_SHORT).show()
                    }
                }

            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    binding.btnSendTest.isEnabled = true
                    binding.btnSendTest.text = if (isGetRequest) "发�?GET 请求" else "发�?POST 请求"
                    Toast.makeText(this@TutorialActivity, "发送失�? ${e.message}", Toast.LENGTH_SHORT).show()
                }
            }
        }
    }
}
