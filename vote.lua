local ip = ARGV[1]
local vote = tonumber(ARGV[2])

local confidenceKey = "confidence"
local numVotesKey = "numVotes"
local ipsKey = "ips"

local confidence = tonumber(redis.call('GET', confidenceKey))
local numVotes = tonumber(redis.call('GET', numVotesKey))
local ipExists = redis.call('HEXISTS', ipsKey, ip)

if ipExists == 1 then
    local oldVote = tonumber(redis.call('HGET', ipsKey, ip))
    if oldVote == vote then
        return {tostring(confidence), tostring(numVotes)}
    end
    confidence = (confidence * numVotes - oldVote + vote) / numVotes
else
    redis.call('INCR', numVotesKey)
    confidence = (confidence * numVotes + vote) / (numVotes + 1)
    numVotes = numVotes + 1
end

redis.call('SET', confidenceKey, tostring(confidence))
redis.call('HSET', ipsKey, ip, tostring(vote))

return {tostring(confidence), tostring(numVotes)}

