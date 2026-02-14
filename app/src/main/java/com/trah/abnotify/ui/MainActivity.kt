package com.trah.abnotify.ui

import android.Manifest
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.view.Menu
import android.view.MenuItem
import android.widget.Toast
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import com.trah.abnotify.AbnotifyApp
import com.trah.abnotify.R
import com.trah.abnotify.databinding.ActivityMainBinding
import com.trah.abnotify.service.WebSocketService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody

class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding
    private val app by lazy { AbnotifyApp.getInstance() }
    
    private val homeFragment = HomeFragment()
    private val messagesFragment = MessagesFragment()
    private val settingsFragment = SettingsFragment()
    private var activeFragment: Fragment = homeFragment

    private val requestPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { isGranted: Boolean ->
        if (isGranted) {
            startWebSocketServiceIfRegistered()
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        setupNavigation()
        checkNotificationPermission()
    }


    private fun setupNavigation() {
        supportFragmentManager.beginTransaction()
            .add(R.id.fragmentContainer, messagesFragment, "messages").hide(messagesFragment)
            .add(R.id.fragmentContainer, settingsFragment, "settings").hide(settingsFragment)
            .add(R.id.fragmentContainer, homeFragment, "home")
            .commit()

        binding.bottomNavigation.setOnItemSelectedListener { item ->
            when (item.itemId) {
                R.id.nav_home -> {
                    if (activeFragment != homeFragment) {
                        supportFragmentManager.beginTransaction().hide(activeFragment).show(homeFragment).commit()
                        activeFragment = homeFragment
                    }
                    true
                }
                R.id.nav_messages -> {
                    if (activeFragment != messagesFragment) {
                        supportFragmentManager.beginTransaction().hide(activeFragment).show(messagesFragment).commit()
                        activeFragment = messagesFragment
                    }
                    true
                }
                R.id.nav_settings -> {
                    if (activeFragment != settingsFragment) {
                        supportFragmentManager.beginTransaction().hide(activeFragment).show(settingsFragment).commit()
                        activeFragment = settingsFragment
                    }
                    true
                }
                else -> false
            }
        }
    }

    override fun onCreateOptionsMenu(menu: Menu): Boolean {
        menuInflater.inflate(R.menu.main_menu, menu)
        return true
    }

    override fun onOptionsItemSelected(item: MenuItem): Boolean {
        return super.onOptionsItemSelected(item)
    }


    fun saveAndRegister() {
        val keyManager = app.keyManager
        val serverUrl = keyManager.serverUrl.trimEnd('/')

        val deviceKey = keyManager.getDeviceKey() ?: return

        CoroutineScope(Dispatchers.IO).launch {
            try {
                val client = OkHttpClient()
                val requestBody = mapOf(
                    "device_key" to deviceKey,
                    "name" to Build.MODEL
                )
                val gson = com.google.gson.Gson()
                val json = gson.toJson(requestBody)
                val request = Request.Builder()
                    .url("$serverUrl/register")
                    .post(json.toRequestBody("application/json".toMediaType()))
                    .build()

                val response = client.newCall(request).execute()

                withContext(Dispatchers.Main) {
                    if (response.isSuccessful) {
                        keyManager.isRegistered = true
                        Toast.makeText(this@MainActivity, "注册成功！", Toast.LENGTH_SHORT).show()
                        restartWebSocketService()
                    } else {
                        Toast.makeText(this@MainActivity, "注册失败: ${response.code}", Toast.LENGTH_SHORT).show()
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    Toast.makeText(this@MainActivity, "网络错误: ${e.message}", Toast.LENGTH_SHORT).show()
                }
            }
        }
    }

    private fun restartWebSocketService() {
        val stopIntent = Intent(this, WebSocketService::class.java).apply {
            action = WebSocketService.ACTION_DISCONNECT
        }
        stopService(stopIntent)

        val startIntent = Intent(this, WebSocketService::class.java).apply {
            action = WebSocketService.ACTION_CONNECT
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            startForegroundService(startIntent)
        } else {
            startService(startIntent)
        }
    }

    private fun checkNotificationPermission() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) {
                requestPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
            } else {
                startWebSocketServiceIfRegistered()
            }
        } else {
            startWebSocketServiceIfRegistered()
        }
    }

    private fun startWebSocketServiceIfRegistered() {
        if (app.keyManager.isRegistered) {
            val intent = Intent(this, WebSocketService::class.java).apply {
                action = WebSocketService.ACTION_CONNECT
            }
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                startForegroundService(intent)
            } else {
                startService(intent)
            }
        }
    }
}
