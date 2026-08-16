param(
    [Parameter(Mandatory = $true)]
    [string]$ResumePath,
    [string]$ApiBase = "http://localhost:8080/api/v1",
    [string]$Username = ("demo_" + (Get-Date -Format "yyyyMMddHHmmss")),
    [string]$Password = "DemoPassword123"
)

$ErrorActionPreference = "Stop"
$resumeFile = Get-Item -LiteralPath $ResumePath

function Send-ResumeFile {
    param(
        [string]$Uri,
        [string]$Token,
        [System.IO.FileInfo]$File
    )

    Add-Type -AssemblyName System.Net.Http
    $client = New-Object System.Net.Http.HttpClient
    $content = New-Object System.Net.Http.MultipartFormDataContent
    $stream = $null
    $fileContent = $null
    try {
        $client.DefaultRequestHeaders.Authorization = New-Object System.Net.Http.Headers.AuthenticationHeaderValue("Bearer", $Token)
        $stream = [System.IO.File]::OpenRead($File.FullName)
        $fileContent = New-Object System.Net.Http.StreamContent($stream)
        $fileContent.Headers.ContentType = New-Object System.Net.Http.Headers.MediaTypeHeaderValue("application/pdf")
        $content.Add($fileContent, "resume", $File.Name)
        $response = $client.PostAsync($Uri, $content).GetAwaiter().GetResult()
        $responseBody = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
        if (-not $response.IsSuccessStatusCode) {
            throw "Resume upload failed with HTTP $([int]$response.StatusCode): $responseBody"
        }
        return $responseBody | ConvertFrom-Json
    } finally {
        if ($fileContent) { $fileContent.Dispose() }
        if ($stream) { $stream.Dispose() }
        $content.Dispose()
        $client.Dispose()
    }
}

Invoke-RestMethod -Method Post -Uri "$ApiBase/auth/register" -ContentType "application/json" -Body (@{
    username = $Username
    password = $Password
} | ConvertTo-Json) | Out-Null

$login = Invoke-RestMethod -Method Post -Uri "$ApiBase/auth/login" -ContentType "application/json" -Body (@{
    username = $Username
    password = $Password
} | ConvertTo-Json)
$headers = @{ Authorization = "Bearer $($login.data.accessToken)" }

$created = Invoke-RestMethod -Method Post -Uri "$ApiBase/interviews" -Headers $headers
$sessionId = $created.data.sessionId
$uploaded = Send-ResumeFile -Uri "$ApiBase/interviews/$sessionId/resume" -Token $login.data.accessToken -File $resumeFile
$jobId = $uploaded.data.jobId

do {
    Start-Sleep -Seconds 1
    $job = Invoke-RestMethod -Method Get -Uri "$ApiBase/jobs/$jobId" -Headers $headers
    Write-Host "Analysis: $($job.data.progress)% $($job.data.stage)"
} while ($job.data.status -in @("QUEUED", "PROCESSING"))

if ($job.data.status -ne "COMPLETED") {
    throw "Analysis failed: $($job.data.errorCode) $($job.data.errorMessage)"
}

for ($turn = 1; $turn -le 30; $turn++) {
    $state = Invoke-RestMethod -Method Get -Uri "$ApiBase/interviews/$sessionId/state" -Headers $headers
    if ($state.data.status -eq "COMPLETED") { break }
    $question = $state.data.currentQuestion
    if (-not $question) { throw "No current question in active session" }
    Write-Host "Q: $($question.content)"
    $answerHeaders = @{
        Authorization = $headers.Authorization
        "Idempotency-Key" = [guid]::NewGuid().ToString()
    }
    $answer = Invoke-RestMethod -Method Post -Uri "$ApiBase/interviews/$sessionId/answers" -Headers $answerHeaders -ContentType "application/json" -Body (@{
        questionNumber = $question.number
        answerContent = "I will explain the project context, my responsibilities, design tradeoffs, failure handling, validation approach, and measurable results."
    } | ConvertTo-Json)
    Write-Host "Score: $($answer.data.score); $($answer.data.feedback)"
    if ($answer.data.finished) { break }
}

$report = Invoke-RestMethod -Method Post -Uri "$ApiBase/interviews/$sessionId/finish" -Headers $headers
Write-Host "Completed. Session: $sessionId; Composite score: $($report.data.compositeScore)"
