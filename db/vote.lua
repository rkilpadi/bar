local ip = ARGV[1]
local vote = tonumber(ARGV[2])

local confidenceKey = "confidence"
local voteCountKey = "voteCount"
local ipsKey = "ips"

local confidence = tonumber(redis.call('GET', confidenceKey))
local voteCount = tonumber(redis.call('GET', voteCountKey))
local ipExists = redis.call('HEXISTS', ipsKey, ip)

if ipExists == 1 then
    local oldVote = tonumber(redis.call('HGET', ipsKey, ip))
    if oldVote == vote then
        return {tostring(confidence), tostring(voteCount)}
    end
    confidence = (confidence * voteCount - oldVote + vote) / voteCount
else
    redis.call('INCR', voteCountKey)
    confidence = (confidence * voteCount + vote) / (voteCount + 1)
    voteCount = voteCount + 1
end

redis.call('SET', confidenceKey, tostring(confidence))
redis.call('HSET', ipsKey, ip, tostring(vote))

return {tostring(confidence), tostring(voteCount)}

